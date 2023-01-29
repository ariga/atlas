-- atlas:txmode none

-- Create a table.
CREATE TABLE t1 (a INTEGER PRIMARY KEY);

-- Cause migrations to fail.
INSERT INTO t1 VALUES (1), (1);
