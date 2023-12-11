-- delimiter should not conflict with compound statements scanning.

create function `add2` (a int, b int) returns int deterministic no sql return a + b;
create function `add3` (a int, b int, c int) returns int deterministic no sql return a + b + c;
delimiter |
-- error 1418
create function fn1(x int) returns int deterministic
begin
       insert into t1 values (x);
       return x+2;
end|
create function fn2(x int) returns int deterministic
begin
       insert into t1 values (x);
       return x+2;
end|
delimiter ;
create function `add4` (a int, b int, c int, d int) returns int deterministic no sql return a + b + c + d;