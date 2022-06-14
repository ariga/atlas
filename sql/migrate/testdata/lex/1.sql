CREATE TABLE t1(id int);


CREATE TABLE t2(id int);

CREATE TABLE t3(id int);

CREATE TABLE t4(
    id int,
    name varchar(255)
);

CREATE TABLE t4(
    id int,
    `name` varchar(255) DEFAULT ';'
) ENGINE=InnoDB;

CREATE TABLE t5(
    id int
    /* comment */
    -- comment
) ENGINE=InnoDB;