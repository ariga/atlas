
CREATE PROCEDURE CompProc(IN p INT)
BEGIN
    DECLARE v1 INT DEFAULT 0;
    DECLARE v2 INT;
    DECLARE d INT DEFAULT FALSE;
    DECLARE c1 CURSOR FOR SELECT a FROM t1 WHERE b = p;
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET d = TRUE;

    OPEN c1;
    rl: LOOP
        FETCH c1 INTO v2;
        IF d THEN
            LEAVE rl;
        END IF;

        BEGIN
            IF v2 > 0 THEN
                UPDATE t2 SET c = c + v2 WHERE d = p;
            END IF;
        END;
    END LOOP;
    CLOSE c1;

    BEGIN
        SELECT COUNT(*) INTO v1 FROM t2 WHERE d = p;
        IF v1 > 100 THEN
            CALL OtherProc(p);
        END IF;
    END;

END;


CREATE FUNCTION CompFunc(eID INT) RETURNS INT
BEGIN
    DECLARE ts INT;
    DECLARE b INT;

    BEGIN
        SELECT SUM(s) INTO ts FROM sales WHERE e = eID;
    END;

    BEGIN
        IF ts > 10000 THEN
            SET b = 500;
        ELSEIF ts BETWEEN 5000 AND 10000 THEN
            SET b = 300;
        ELSE
            SET b = 0;
        END IF;
    END;

    RETURN b;
END;


CREATE PROCEDURE CompLogicProc()
BEGIN
    DECLARE i INT DEFAULT 1;
    DECLARE t INT DEFAULT 0;

    sl: WHILE i <= 10 DO
        BEGIN
            IF i MOD 2 = 0 THEN
                SET t = t + i;
            ELSE
                BEGIN
                    IF i MOD 3 = 0 THEN
                        SET t = t - i;
                    END IF;
                END;
            END IF;
            SET i = i + 1;
        END;
    END WHILE;

    BEGIN
        IF t < 0 THEN
            SET t = 0;
        END IF;
    END;

    SELECT t;
END;


CREATE PROCEDURE ff1()
BEGIN
    DECLARE v1, v2, v3, i, j INT DEFAULT 0;
    DECLARE flag INT DEFAULT FALSE;
    DECLARE cur CURSOR FOR SELECT col1 FROM tableX;
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET flag = TRUE;

    OPEN cur;
    main_loop: LOOP
        FETCH cur INTO v1;
        IF flag THEN
            LEAVE main_loop;
        END IF;

        SET i = 1;
        WHILE i <= 5 DO
            SET j = 1;
            REPEAT
                SET v2 = (SELECT COUNT(*) FROM tableY WHERE col2 = v1 AND col3 = j);
                IF v2 > 3 THEN
                    UPDATE tableZ SET col4 = col4 + 1 WHERE col5 = i;
                END IF;
                SET j = j + 1;
            UNTIL j > 3 END REPEAT;
            SET i = i + 1;
        END WHILE;

        compound_logic: BEGIN
            IF v1 < 10 THEN
                INSERT INTO tableA (colA) VALUES (v1);
            ELSEIF v1 BETWEEN 10 AND 20 THEN
                UPDATE tableB SET colB = v1 WHERE colB < v1;
            ELSE
                DELETE FROM tableC WHERE colC = v1;
            END IF;
        END;
    END LOOP;
    CLOSE cur;
END;