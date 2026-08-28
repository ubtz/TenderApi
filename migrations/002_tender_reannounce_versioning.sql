SET XACT_ABORT ON;
BEGIN TRANSACTION;

/*
    Tender version lifecycle
    ------------------------
    Active -> WaitingExternal -> Archived
    A returned external package creates the next active Tender version.
*/

IF COL_LENGTH('Tender.dbo.Tender', 'VersionNo') IS NULL
    ALTER TABLE [Tender].[dbo].[Tender] ADD [VersionNo] INT NULL;

IF COL_LENGTH('Tender.dbo.Tender', 'IsCurrent') IS NULL
    ALTER TABLE [Tender].[dbo].[Tender] ADD [IsCurrent] BIT NULL;

IF COL_LENGTH('Tender.dbo.Tender', 'LifecycleStatus') IS NULL
    ALTER TABLE [Tender].[dbo].[Tender] ADD [LifecycleStatus] NVARCHAR(30) NULL;

IF COL_LENGTH('Tender.dbo.Tender', 'ArchivedAt') IS NULL
    ALTER TABLE [Tender].[dbo].[Tender] ADD [ArchivedAt] DATETIME2 NULL;

;WITH VersionChain AS (
    SELECT [TenderId], CAST(1 AS INT) AS [VersionNo]
    FROM [Tender].[dbo].[Tender]
    WHERE [RootTenderId] IS NULL

    UNION ALL

    SELECT child.[TenderId], parent.[VersionNo] + 1
    FROM [Tender].[dbo].[Tender] child
    INNER JOIN VersionChain parent ON child.[RootTenderId] = parent.[TenderId]
)
UPDATE tender
SET tender.[VersionNo] = chain.[VersionNo]
FROM [Tender].[dbo].[Tender] tender
INNER JOIN VersionChain chain ON chain.[TenderId] = tender.[TenderId]
OPTION (MAXRECURSION 32767);

UPDATE tender
SET
    [VersionNo] = ISNULL([VersionNo], 1),
    [IsCurrent] = CASE
        WHEN ISNULL(tender.[IsDeleted], 0) = 1 THEN 0
        WHEN EXISTS (
            SELECT 1
            FROM [Tender].[dbo].[Tender] child
            WHERE child.[RootTenderId] = tender.[TenderId]
              AND ISNULL(child.[IsDeleted], 0) = 0
        ) THEN 0
        ELSE 1
    END,
    [LifecycleStatus] = CASE
        WHEN ISNULL(tender.[IsDeleted], 0) = 1 THEN N'Deleted'
        WHEN EXISTS (
            SELECT 1
            FROM [Tender].[dbo].[Tender] child
            WHERE child.[RootTenderId] = tender.[TenderId]
              AND ISNULL(child.[IsDeleted], 0) = 0
        ) THEN N'Archived'
        ELSE N'Active'
    END
FROM [Tender].[dbo].[Tender] tender;

ALTER TABLE [Tender].[dbo].[Tender] ALTER COLUMN [VersionNo] INT NOT NULL;
ALTER TABLE [Tender].[dbo].[Tender] ALTER COLUMN [IsCurrent] BIT NOT NULL;
ALTER TABLE [Tender].[dbo].[Tender] ALTER COLUMN [LifecycleStatus] NVARCHAR(30) NOT NULL;

IF NOT EXISTS (
    SELECT 1 FROM [Tender].sys.default_constraints dc
    INNER JOIN [Tender].sys.columns col
        ON col.[object_id] = dc.[parent_object_id]
       AND col.[column_id] = dc.[parent_column_id]
    WHERE dc.[parent_object_id] = OBJECT_ID('[Tender].[dbo].[Tender]')
      AND col.[name] = 'VersionNo'
)
    ALTER TABLE [Tender].[dbo].[Tender]
        ADD CONSTRAINT [DF_Tender_VersionNo] DEFAULT (1) FOR [VersionNo];

IF NOT EXISTS (
    SELECT 1 FROM [Tender].sys.default_constraints dc
    INNER JOIN [Tender].sys.columns col
        ON col.[object_id] = dc.[parent_object_id]
       AND col.[column_id] = dc.[parent_column_id]
    WHERE dc.[parent_object_id] = OBJECT_ID('[Tender].[dbo].[Tender]')
      AND col.[name] = 'IsCurrent'
)
    ALTER TABLE [Tender].[dbo].[Tender]
        ADD CONSTRAINT [DF_Tender_IsCurrent] DEFAULT (1) FOR [IsCurrent];

IF NOT EXISTS (
    SELECT 1 FROM [Tender].sys.default_constraints dc
    INNER JOIN [Tender].sys.columns col
        ON col.[object_id] = dc.[parent_object_id]
       AND col.[column_id] = dc.[parent_column_id]
    WHERE dc.[parent_object_id] = OBJECT_ID('[Tender].[dbo].[Tender]')
      AND col.[name] = 'LifecycleStatus'
)
    ALTER TABLE [Tender].[dbo].[Tender]
        ADD CONSTRAINT [DF_Tender_LifecycleStatus] DEFAULT (N'Active') FOR [LifecycleStatus];

IF NOT EXISTS (
    SELECT 1 FROM [Tender].sys.indexes
    WHERE [name] = 'IX_Tender_CurrentLifecycle'
      AND [object_id] = OBJECT_ID('[Tender].[dbo].[Tender]')
)
    CREATE INDEX [IX_Tender_CurrentLifecycle]
        ON [Tender].[dbo].[Tender] ([IsCurrent], [LifecycleStatus], [CreatedAt]);

IF OBJECT_ID('[Tender].[dbo].[TenderVersionItems]') IS NULL
BEGIN
    CREATE TABLE [Tender].[dbo].[TenderVersionItems] (
        [Id] BIGINT IDENTITY(1,1) NOT NULL PRIMARY KEY,
        [TenderId] INT NOT NULL,
        [BasketId] INT NULL,
        [PkgNo] NVARCHAR(100) NOT NULL,
        [PkgDate] DATE NOT NULL,
        [PkgName] NVARCHAR(150) NOT NULL,
        [Code] NVARCHAR(50) NOT NULL,
        [Decision] TINYINT NOT NULL CONSTRAINT [DF_TenderVersionItems_Decision] DEFAULT (0),
        [DecidedByUserId] INT NULL,
        [DecidedAt] DATETIME2 NULL,
        [CreatedAt] DATETIME2 NOT NULL CONSTRAINT [DF_TenderVersionItems_CreatedAt] DEFAULT (SYSDATETIME()),
        CONSTRAINT [FK_TenderVersionItems_Tender] FOREIGN KEY ([TenderId])
            REFERENCES [Tender].[dbo].[Tender] ([TenderId]),
        CONSTRAINT [CK_TenderVersionItems_Decision] CHECK ([Decision] IN (0, 1, 2))
    );

    CREATE UNIQUE INDEX [UX_TenderVersionItems_Result]
        ON [Tender].[dbo].[TenderVersionItems]
        ([TenderId], [BasketId], [PkgNo], [PkgDate], [Code]);

    CREATE INDEX [IX_TenderVersionItems_TenderDecision]
        ON [Tender].[dbo].[TenderVersionItems] ([TenderId], [Decision]);
END;

IF OBJECT_ID('[Tender].[dbo].[TenderRecycleBatch]') IS NULL
BEGIN
    CREATE TABLE [Tender].[dbo].[TenderRecycleBatch] (
        [Id] BIGINT IDENTITY(1,1) NOT NULL PRIMARY KEY,
        [SourceTenderId] INT NOT NULL,
        [SourceVersionNo] INT NOT NULL,
        [NextTenderId] INT NULL,
        [Status] NVARCHAR(30) NOT NULL CONSTRAINT [DF_TenderRecycleBatch_Status] DEFAULT (N'WaitingExternal'),
        [CreatedByUserId] INT NOT NULL,
        [CreatedAt] DATETIME2 NOT NULL CONSTRAINT [DF_TenderRecycleBatch_CreatedAt] DEFAULT (SYSDATETIME()),
        [CompletedAt] DATETIME2 NULL,
        CONSTRAINT [FK_TenderRecycleBatch_SourceTender] FOREIGN KEY ([SourceTenderId])
            REFERENCES [Tender].[dbo].[Tender] ([TenderId]),
        CONSTRAINT [FK_TenderRecycleBatch_NextTender] FOREIGN KEY ([NextTenderId])
            REFERENCES [Tender].[dbo].[Tender] ([TenderId]),
        CONSTRAINT [CK_TenderRecycleBatch_Status]
            CHECK ([Status] IN (N'WaitingExternal', N'Matching', N'Completed', N'Failed', N'Cancelled'))
    );

    CREATE INDEX [IX_TenderRecycleBatch_Status]
        ON [Tender].[dbo].[TenderRecycleBatch] ([Status], [CreatedAt]);
END;

IF OBJECT_ID('[Tender].[dbo].[TenderRecyclePackages]') IS NULL
BEGIN
    CREATE TABLE [Tender].[dbo].[TenderRecyclePackages] (
        [Id] BIGINT IDENTITY(1,1) NOT NULL PRIMARY KEY,
        [RecycleBatchId] BIGINT NOT NULL,
        [PkgNo] NVARCHAR(100) NOT NULL,
        [PkgDate] DATE NOT NULL,
        [PkgName] NVARCHAR(150) NOT NULL,
        [Status] NVARCHAR(20) NOT NULL CONSTRAINT [DF_TenderRecyclePackages_Status] DEFAULT (N'Waiting'),
        [MatchedAt] DATETIME2 NULL,
        CONSTRAINT [FK_TenderRecyclePackages_Batch] FOREIGN KEY ([RecycleBatchId])
            REFERENCES [Tender].[dbo].[TenderRecycleBatch] ([Id]),
        CONSTRAINT [CK_TenderRecyclePackages_Status]
            CHECK ([Status] IN (N'Waiting', N'Matched', N'Completed', N'Cancelled'))
    );

    CREATE UNIQUE INDEX [UX_TenderRecyclePackages_Identity]
        ON [Tender].[dbo].[TenderRecyclePackages]
        ([RecycleBatchId], [PkgNo], [PkgDate], [PkgName]);

    CREATE INDEX [IX_TenderRecyclePackages_Match]
        ON [Tender].[dbo].[TenderRecyclePackages]
        ([Status], [PkgDate], [PkgNo], [PkgName]);
END;

IF OBJECT_ID('[Tender].[dbo].[TenderRecycleCodes]') IS NULL
BEGIN
    CREATE TABLE [Tender].[dbo].[TenderRecycleCodes] (
        [Id] BIGINT IDENTITY(1,1) NOT NULL PRIMARY KEY,
        [RecyclePackageId] BIGINT NOT NULL,
        [Code] NVARCHAR(50) NOT NULL,
        [MatchedAt] DATETIME2 NULL,
        CONSTRAINT [FK_TenderRecycleCodes_Package] FOREIGN KEY ([RecyclePackageId])
            REFERENCES [Tender].[dbo].[TenderRecyclePackages] ([Id])
    );

    CREATE UNIQUE INDEX [UX_TenderRecycleCodes_PackageCode]
        ON [Tender].[dbo].[TenderRecycleCodes] ([RecyclePackageId], [Code]);
END;

IF OBJECT_ID('[Tender].[dbo].[TenderRecycleOutbox]') IS NULL
BEGIN
    CREATE TABLE [Tender].[dbo].[TenderRecycleOutbox] (
        [Id] BIGINT IDENTITY(1,1) NOT NULL PRIMARY KEY,
        [RecycleBatchId] BIGINT NOT NULL,
        [Payload] NVARCHAR(MAX) NOT NULL,
        [Status] NVARCHAR(20) NOT NULL CONSTRAINT [DF_TenderRecycleOutbox_Status] DEFAULT (N'Pending'),
        [AttemptCount] INT NOT NULL CONSTRAINT [DF_TenderRecycleOutbox_AttemptCount] DEFAULT (0),
        [LastError] NVARCHAR(1000) NULL,
        [CreatedAt] DATETIME2 NOT NULL CONSTRAINT [DF_TenderRecycleOutbox_CreatedAt] DEFAULT (SYSDATETIME()),
        [SentAt] DATETIME2 NULL,
        CONSTRAINT [FK_TenderRecycleOutbox_Batch] FOREIGN KEY ([RecycleBatchId])
            REFERENCES [Tender].[dbo].[TenderRecycleBatch] ([Id]),
        CONSTRAINT [CK_TenderRecycleOutbox_Status]
            CHECK ([Status] IN (N'Pending', N'Sending', N'Sent', N'Failed'))
    );

    CREATE UNIQUE INDEX [UX_TenderRecycleOutbox_Batch]
        ON [Tender].[dbo].[TenderRecycleOutbox] ([RecycleBatchId]);
END;

/* Production/log schema mirrors the development schema. */
IF OBJECT_ID('[Tender].[logtender].[Tender]') IS NOT NULL
BEGIN
    IF COL_LENGTH('Tender.logtender.Tender', 'VersionNo') IS NULL
        ALTER TABLE [Tender].[logtender].[Tender] ADD [VersionNo] INT NULL;

    IF COL_LENGTH('Tender.logtender.Tender', 'IsCurrent') IS NULL
        ALTER TABLE [Tender].[logtender].[Tender] ADD [IsCurrent] BIT NULL;

    IF COL_LENGTH('Tender.logtender.Tender', 'LifecycleStatus') IS NULL
        ALTER TABLE [Tender].[logtender].[Tender] ADD [LifecycleStatus] NVARCHAR(30) NULL;

    IF COL_LENGTH('Tender.logtender.Tender', 'ArchivedAt') IS NULL
        ALTER TABLE [Tender].[logtender].[Tender] ADD [ArchivedAt] DATETIME2 NULL;

    ;WITH VersionChain AS (
        SELECT [TenderId], CAST(1 AS INT) AS [VersionNo]
        FROM [Tender].[logtender].[Tender]
        WHERE [RootTenderId] IS NULL

        UNION ALL

        SELECT child.[TenderId], parent.[VersionNo] + 1
        FROM [Tender].[logtender].[Tender] child
        INNER JOIN VersionChain parent ON child.[RootTenderId] = parent.[TenderId]
    )
    UPDATE tender
    SET tender.[VersionNo] = chain.[VersionNo]
    FROM [Tender].[logtender].[Tender] tender
    INNER JOIN VersionChain chain ON chain.[TenderId] = tender.[TenderId]
    OPTION (MAXRECURSION 32767);

    UPDATE tender
    SET
        [VersionNo] = ISNULL([VersionNo], 1),
        [IsCurrent] = CASE
            WHEN ISNULL(tender.[IsDeleted], 0) = 1 THEN 0
            WHEN EXISTS (
                SELECT 1 FROM [Tender].[logtender].[Tender] child
                WHERE child.[RootTenderId] = tender.[TenderId]
                  AND ISNULL(child.[IsDeleted], 0) = 0
            ) THEN 0 ELSE 1 END,
        [LifecycleStatus] = CASE
            WHEN ISNULL(tender.[IsDeleted], 0) = 1 THEN N'Deleted'
            WHEN EXISTS (
                SELECT 1 FROM [Tender].[logtender].[Tender] child
                WHERE child.[RootTenderId] = tender.[TenderId]
                  AND ISNULL(child.[IsDeleted], 0) = 0
            ) THEN N'Archived' ELSE N'Active' END
    FROM [Tender].[logtender].[Tender] tender;

    ALTER TABLE [Tender].[logtender].[Tender] ALTER COLUMN [VersionNo] INT NOT NULL;
    ALTER TABLE [Tender].[logtender].[Tender] ALTER COLUMN [IsCurrent] BIT NOT NULL;
    ALTER TABLE [Tender].[logtender].[Tender] ALTER COLUMN [LifecycleStatus] NVARCHAR(30) NOT NULL;

    IF NOT EXISTS (
        SELECT 1 FROM [Tender].sys.default_constraints dc
        INNER JOIN [Tender].sys.columns col
            ON col.[object_id] = dc.[parent_object_id]
           AND col.[column_id] = dc.[parent_column_id]
        WHERE dc.[parent_object_id] = OBJECT_ID('[Tender].[logtender].[Tender]')
          AND col.[name] = 'VersionNo'
    )
        ALTER TABLE [Tender].[logtender].[Tender]
            ADD CONSTRAINT [DF_Tender_VersionNo] DEFAULT (1) FOR [VersionNo];

    IF NOT EXISTS (
        SELECT 1 FROM [Tender].sys.default_constraints dc
        INNER JOIN [Tender].sys.columns col
            ON col.[object_id] = dc.[parent_object_id]
           AND col.[column_id] = dc.[parent_column_id]
        WHERE dc.[parent_object_id] = OBJECT_ID('[Tender].[logtender].[Tender]')
          AND col.[name] = 'IsCurrent'
    )
        ALTER TABLE [Tender].[logtender].[Tender]
            ADD CONSTRAINT [DF_Tender_IsCurrent] DEFAULT (1) FOR [IsCurrent];

    IF NOT EXISTS (
        SELECT 1 FROM [Tender].sys.default_constraints dc
        INNER JOIN [Tender].sys.columns col
            ON col.[object_id] = dc.[parent_object_id]
           AND col.[column_id] = dc.[parent_column_id]
        WHERE dc.[parent_object_id] = OBJECT_ID('[Tender].[logtender].[Tender]')
          AND col.[name] = 'LifecycleStatus'
    )
        ALTER TABLE [Tender].[logtender].[Tender]
            ADD CONSTRAINT [DF_Tender_LifecycleStatus] DEFAULT (N'Active') FOR [LifecycleStatus];

    IF NOT EXISTS (
        SELECT 1 FROM [Tender].sys.indexes
        WHERE [name] = 'IX_Tender_CurrentLifecycle'
          AND [object_id] = OBJECT_ID('[Tender].[logtender].[Tender]')
    )
        CREATE INDEX [IX_Tender_CurrentLifecycle]
            ON [Tender].[logtender].[Tender] ([IsCurrent], [LifecycleStatus], [CreatedAt]);

    IF OBJECT_ID('[Tender].[logtender].[TenderVersionItems]') IS NULL
    BEGIN
        CREATE TABLE [Tender].[logtender].[TenderVersionItems] (
            [Id] BIGINT IDENTITY(1,1) NOT NULL PRIMARY KEY,
            [TenderId] INT NOT NULL,
            [BasketId] INT NULL,
            [PkgNo] NVARCHAR(100) NOT NULL,
            [PkgDate] DATE NOT NULL,
            [PkgName] NVARCHAR(150) NOT NULL,
            [Code] NVARCHAR(50) NOT NULL,
            [Decision] TINYINT NOT NULL DEFAULT (0),
            [DecidedByUserId] INT NULL,
            [DecidedAt] DATETIME2 NULL,
            [CreatedAt] DATETIME2 NOT NULL DEFAULT (SYSDATETIME()),
            CONSTRAINT [FK_TenderVersionItems_Tender] FOREIGN KEY ([TenderId])
                REFERENCES [Tender].[logtender].[Tender] ([TenderId]),
            CONSTRAINT [CK_TenderVersionItems_Decision] CHECK ([Decision] IN (0, 1, 2))
        );
        CREATE UNIQUE INDEX [UX_TenderVersionItems_Result]
            ON [Tender].[logtender].[TenderVersionItems]
            ([TenderId], [BasketId], [PkgNo], [PkgDate], [Code]);
        CREATE INDEX [IX_TenderVersionItems_TenderDecision]
            ON [Tender].[logtender].[TenderVersionItems] ([TenderId], [Decision]);
    END;

    IF OBJECT_ID('[Tender].[logtender].[TenderRecycleBatch]') IS NULL
    BEGIN
        CREATE TABLE [Tender].[logtender].[TenderRecycleBatch] (
            [Id] BIGINT IDENTITY(1,1) NOT NULL PRIMARY KEY,
            [SourceTenderId] INT NOT NULL,
            [SourceVersionNo] INT NOT NULL,
            [NextTenderId] INT NULL,
            [Status] NVARCHAR(30) NOT NULL DEFAULT (N'WaitingExternal'),
            [CreatedByUserId] INT NOT NULL,
            [CreatedAt] DATETIME2 NOT NULL DEFAULT (SYSDATETIME()),
            [CompletedAt] DATETIME2 NULL,
            CONSTRAINT [FK_TenderRecycleBatch_SourceTender] FOREIGN KEY ([SourceTenderId])
                REFERENCES [Tender].[logtender].[Tender] ([TenderId]),
            CONSTRAINT [FK_TenderRecycleBatch_NextTender] FOREIGN KEY ([NextTenderId])
                REFERENCES [Tender].[logtender].[Tender] ([TenderId]),
            CONSTRAINT [CK_TenderRecycleBatch_Status]
                CHECK ([Status] IN (N'WaitingExternal', N'Matching', N'Completed', N'Failed', N'Cancelled'))
        );
        CREATE INDEX [IX_TenderRecycleBatch_Status]
            ON [Tender].[logtender].[TenderRecycleBatch] ([Status], [CreatedAt]);
    END;

    IF OBJECT_ID('[Tender].[logtender].[TenderRecyclePackages]') IS NULL
    BEGIN
        CREATE TABLE [Tender].[logtender].[TenderRecyclePackages] (
            [Id] BIGINT IDENTITY(1,1) NOT NULL PRIMARY KEY,
            [RecycleBatchId] BIGINT NOT NULL,
            [PkgNo] NVARCHAR(100) NOT NULL,
            [PkgDate] DATE NOT NULL,
            [PkgName] NVARCHAR(150) NOT NULL,
            [Status] NVARCHAR(20) NOT NULL DEFAULT (N'Waiting'),
            [MatchedAt] DATETIME2 NULL,
            CONSTRAINT [FK_TenderRecyclePackages_Batch] FOREIGN KEY ([RecycleBatchId])
                REFERENCES [Tender].[logtender].[TenderRecycleBatch] ([Id]),
            CONSTRAINT [CK_TenderRecyclePackages_Status]
                CHECK ([Status] IN (N'Waiting', N'Matched', N'Completed', N'Cancelled'))
        );
        CREATE UNIQUE INDEX [UX_TenderRecyclePackages_Identity]
            ON [Tender].[logtender].[TenderRecyclePackages]
            ([RecycleBatchId], [PkgNo], [PkgDate], [PkgName]);
        CREATE INDEX [IX_TenderRecyclePackages_Match]
            ON [Tender].[logtender].[TenderRecyclePackages]
            ([Status], [PkgDate], [PkgNo], [PkgName]);
    END;

    IF OBJECT_ID('[Tender].[logtender].[TenderRecycleCodes]') IS NULL
    BEGIN
        CREATE TABLE [Tender].[logtender].[TenderRecycleCodes] (
            [Id] BIGINT IDENTITY(1,1) NOT NULL PRIMARY KEY,
            [RecyclePackageId] BIGINT NOT NULL,
            [Code] NVARCHAR(50) NOT NULL,
            [MatchedAt] DATETIME2 NULL,
            CONSTRAINT [FK_TenderRecycleCodes_Package] FOREIGN KEY ([RecyclePackageId])
                REFERENCES [Tender].[logtender].[TenderRecyclePackages] ([Id])
        );
        CREATE UNIQUE INDEX [UX_TenderRecycleCodes_PackageCode]
            ON [Tender].[logtender].[TenderRecycleCodes] ([RecyclePackageId], [Code]);
    END;

    IF OBJECT_ID('[Tender].[logtender].[TenderRecycleOutbox]') IS NULL
    BEGIN
        CREATE TABLE [Tender].[logtender].[TenderRecycleOutbox] (
            [Id] BIGINT IDENTITY(1,1) NOT NULL PRIMARY KEY,
            [RecycleBatchId] BIGINT NOT NULL,
            [Payload] NVARCHAR(MAX) NOT NULL,
            [Status] NVARCHAR(20) NOT NULL DEFAULT (N'Pending'),
            [AttemptCount] INT NOT NULL DEFAULT (0),
            [LastError] NVARCHAR(1000) NULL,
            [CreatedAt] DATETIME2 NOT NULL DEFAULT (SYSDATETIME()),
            [SentAt] DATETIME2 NULL,
            CONSTRAINT [FK_TenderRecycleOutbox_Batch] FOREIGN KEY ([RecycleBatchId])
                REFERENCES [Tender].[logtender].[TenderRecycleBatch] ([Id]),
            CONSTRAINT [CK_TenderRecycleOutbox_Status]
                CHECK ([Status] IN (N'Pending', N'Sending', N'Sent', N'Failed'))
        );
        CREATE UNIQUE INDEX [UX_TenderRecycleOutbox_Batch]
            ON [Tender].[logtender].[TenderRecycleOutbox] ([RecycleBatchId]);
    END;
END;

COMMIT TRANSACTION;
