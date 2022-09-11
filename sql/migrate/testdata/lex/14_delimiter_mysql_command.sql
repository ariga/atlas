DELIMITER //
CREATE PROCEDURE dorepeat(p1 INT)
    BEGIN
    SET @x = 0;
    REPEAT SET @x = @x + 1; UNTIL @x > p1 END REPEAT;
END
//
DELIMITER ;
CALL dorepeat(100)