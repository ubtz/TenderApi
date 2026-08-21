SET XACT_ABORT ON;
BEGIN TRANSACTION;

IF COL_LENGTH('Tender.dbo.Users', 'PasswordHash') IS NOT NULL
    ALTER TABLE [Tender].[dbo].[Users] ALTER COLUMN [PasswordHash] NVARCHAR(255) NOT NULL;

IF COL_LENGTH('Tender.logtender.Users', 'PasswordHash') IS NOT NULL
    ALTER TABLE [Tender].[logtender].[Users] ALTER COLUMN [PasswordHash] NVARCHAR(255) NOT NULL;

IF EXISTS (
    SELECT 1
    FROM [Tender].[dbo].[Basket]
    WHERE ISNULL(IsTemp, 0) = 1
    GROUP BY UserId
    HAVING COUNT(*) > 1
)
    THROW 51000, 'Duplicate temporary baskets exist in Tender.dbo.Basket. Resolve them before creating the unique index.', 1;

IF NOT EXISTS (
    SELECT 1
    FROM [Tender].sys.indexes
    WHERE name = 'UX_Basket_OneTempPerUser'
      AND object_id = OBJECT_ID('[Tender].[dbo].[Basket]')
)
    CREATE UNIQUE INDEX [UX_Basket_OneTempPerUser]
        ON [Tender].[dbo].[Basket] ([UserId])
        WHERE [IsTemp] = 1;

IF OBJECT_ID('[Tender].[logtender].[Basket]') IS NOT NULL
   AND EXISTS (
       SELECT 1
       FROM [Tender].[logtender].[Basket]
       WHERE ISNULL(IsTemp, 0) = 1
       GROUP BY UserId
       HAVING COUNT(*) > 1
   )
    THROW 51001, 'Duplicate temporary baskets exist in Tender.logtender.Basket. Resolve them before creating the unique index.', 1;

IF OBJECT_ID('[Tender].[logtender].[Basket]') IS NOT NULL
   AND NOT EXISTS (
       SELECT 1
       FROM [Tender].sys.indexes
       WHERE name = 'UX_Basket_OneTempPerUser'
         AND object_id = OBJECT_ID('[Tender].[logtender].[Basket]')
   )
    CREATE UNIQUE INDEX [UX_Basket_OneTempPerUser]
        ON [Tender].[logtender].[Basket] ([UserId])
        WHERE [IsTemp] = 1;

COMMIT TRANSACTION;
