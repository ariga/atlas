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

select * from t1 where `begin` < 10;

create table t2(begin int);
select * from t2 where begin < 10;

create table t3(begin int, end int);
select * from t3 where begin <> end;

create table t4(begin int, end int);
select * from t4 where begin > end or begin < end or begin = end or begin in (1,2,3) and end not in (1,2,3);

create table t5(begin int, end int);
create trigger t5_insert before insert on t5 for each row set new.begin = new.end;
create trigger t5_insert before insert on t5 for each row
begin
    set new.begin = new.end;
end;

/*
 The "NEW.begin" one is not scanned as a "BEGIN" because it is
 not a beginning of s statement and there is no \s before it.
*/
CREATE TRIGGER begin1 BEFORE INSERT ON t5
FOR EACH ROW
BEGIN
    SELECT 1;
    SELECT 1;
    SET NEW.begin = NEW.end;
END;

/*
 The `begin` column is quoted as an identifier as therefore is skipped.
*/
CREATE TRIGGER begin2 BEFORE INSERT ON t5
    FOR EACH ROW
BEGIN
    SELECT 1;
    SELECT 1;
    SET `begin` = `end`;
END;

/*
 An unquoted begin confuses the lexer, and requires using the DELIMITER command.
*/
-- Set a special statement delimiter.
DELIMITER //;

CREATE TRIGGER begin3 BEFORE INSERT ON t5
    FOR EACH ROW
BEGIN
    SELECT 1;
    SELECT 1;
    SET begin = end;
END //;

-- Unset the special statement delimiter.
DELIMITER ;

-- issue 2397.
CREATE OR REPLACE VIEW claims.current_member_view AS
    SELECT l.member_load_id, n.name_last_or_organization, g.name_given, n.name_middle, n.name_prefix, n.name_suffix, ps.external_plan_sponsor_id, i.member_identification_code, i.date_of_birth, cgop.group_or_policy_number, a.plan_sponsor_name, stat.is_subscriber, stat.relationship_to_subscriber, ed.employment_begin, cd.benefit_begin, cd.benefit_end
    FROM claims.member_coverage_load mcl
    LEFT OUTER JOIN claims.member_load l ON mcl.member_load_id = l.member_load_id
    LEFT OUTER JOIN claims.plan_sponsor_match m ON mcl.coverage_group_or_plan_id = m.coverage_group_or_plan_id
    LEFT OUTER JOIN claims.plan_sponsor ps ON ps.plan_sponsor_id = m.plan_sponsor_id
    LEFT OUTER JOIN claims.member_seen ms ON ms.member_seen_id = l.member_seen_id
    LEFT OUTER JOIN claims.member_identification i ON ms.member_identification_id = i.member_identification_id
    LEFT OUTER JOIN claims.name n ON n.name_id = ms.name_id
    LEFT OUTER JOIN claims.name_given g ON n.name_first_id = g.name_given_id
    LEFT OUTER JOIN claims.eligibility_date ed ON ed.eligibility_date_id = l.eligibility_date_id
    LEFT OUTER JOIN claims.eligibility_reference r ON r.eligibility_reference_id = l.eligibility_reference_id
    LEFT OUTER JOIN claims.eligibility_status stat ON stat.eligibility_status_id = l.eligibility_status_id
    LEFT OUTER JOIN claims.eligibility_admin a ON a.eligibility_admin_id = l.eligibility_admin_id
    LEFT OUTER JOIN claims.eligibility_submission s ON s.eligibility_submission_id = l.eligibility_submission_id
    LEFT OUTER JOIN claims.coverage_group_or_plan cgop ON cgop.coverage_group_or_plan_id = mcl.coverage_group_or_plan_id
    LEFT OUTER JOIN claims.coverage_date cd ON cd.coverage_date_id = mcl.coverage_date_id
    WHERE benefit_begin < NOW() AND benefit_end > NOW();