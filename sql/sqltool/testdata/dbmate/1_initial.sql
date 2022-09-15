-- migrate:up
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

INSERT INTO post (title) VALUES (
'This is
my multiline

value');

-- migrate:down


DROP TABLE post;