-- An example for supporting MySQL client delimiters.

DELIMITER $$

CREATE OR REPLACE FUNCTION gen_uuid() RETURNS VARCHAR(22)
BEGIN
    RETURN concat(
        date_format(NOW(6), '%Y%m%d%i%s%f'),
        ROUND(1 + RAND() * (100 - 2))
    );
END;$$

DELIMITER ;

CALL gen_uuid();
