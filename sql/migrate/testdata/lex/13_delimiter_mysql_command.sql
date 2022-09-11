# Test delimiter :
select "Test delimiter :" as " ";
delimiter :
select * from t1:
/* Delimiter commands can have comments */
delimiter ;
select 'End test :';

/* Test delimiter :; */
select "Test delimiter :;" as " ";
delimiter :;
select * from t1 :;
delimiter ;
select 'End test :;';

-- Test delimiter //
select "Test delimiter //" as " ";
delimiter //
select * from t1//
delimiter ;
select 'End test //';

# Test delimiter 'MySQL'
select "Test delimiter MySQL" as " ";
delimiter 'MySQL'
select * from t1MySQL
delimiter ;
select 'End test MySQL';

# Test delimiter 'delimiter'
select "Test delimiter delimiter" as " ";
delimiter delimiter
select * from t1delimiter
delimiter ;
select 'End test delimiter';

# Test delimiter @@
select "Test delimiter @@" as " ";
delimiter @@
select * from t1 @@
select * from t2@@
alter table t add column c@@
delimiter ;
select 'End test @@';

# Test delimiter \n\n
select "Test delimiter \n\n" as " ";
delimiter \n\n
select * from t1

select * from t2

delimiter ;
select 'End test \\n\\n';