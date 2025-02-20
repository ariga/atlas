CREATE FUNCTION [f2](@a as INT, @b as INT = 1)
RETURNS TABLE
AS RETURN SELECT @a as [a], @b as [b], (@a+@b)*2 as [p], @a*@b as [s];
CREATE FUNCTION [f3] (@a int, @b int = 1) RETURNS @t1 TABLE ([c1] int NOT NULL, [c2] nvarchar(255) COLLATE SQL_Latin1_General_CP1_CI_AS NOT NULL, [c3] nvarchar(255) COLLATE SQL_Latin1_General_CP1_CI_AS DEFAULT N'G' NULL, [c4] int NOT NULL, PRIMARY KEY CLUSTERED ([c1] ASC), INDEX [idx] NONCLUSTERED ([c2] ASC), UNIQUE NONCLUSTERED ([c2] ASC, [c3] DESC), UNIQUE NONCLUSTERED ([c3] DESC, [c4] ASC), CHECK ([c4]>(0))) AS BEGIN 
  INSERT @t1
  SELECT 1 AS [c1], 'A' AS [c2], NULL AS [c3], @a * @a + @b AS [c4];
RETURN
END
