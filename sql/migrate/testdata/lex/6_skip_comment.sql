-- comment 1
CREATE TABLE t1(id int);
# CREATE TABLE t1(id int);

-- comment 2
# CREATE TABLE t2(id int);
CREATE TABLE t2(id int);
-- comment 3
CREATE TABLE t3(id int);

# CREATE TABLE t3(id int);

/* comment 4 */
CREATE TABLE t4(
    id int
    /* comment */
    -- comment
) ENGINE=InnoDB;

/* comment 5
*/
CREATE TABLE t5(
    id int
    /* comment */
    -- comment
) ENGINE=InnoDB;
