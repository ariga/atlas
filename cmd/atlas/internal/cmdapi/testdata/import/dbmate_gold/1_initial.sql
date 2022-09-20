CREATE TABLE post
(
    id    int NOT NULL,
    title text,
    body  text,
    PRIMARY KEY (id)
);
/*
 Multiline comment ...
 */
ALTER TABLE post ADD created_at TIMESTAMP NOT NULL;
-- Normal comment
-- With a second line
INSERT INTO post (title) VALUES (
'This is
my multiline

value');
