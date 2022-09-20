-- comment
CREATE TABLE post
(
    id    int NOT NULL,
    title text,
    body  text,
    PRIMARY KEY (id)
);

ALTER TABLE post ADD created_at TIMESTAMP NOT NULL;

INSERT INTO post (title, created_at) VALUES (
'This is
my multiline

value', NOW());
