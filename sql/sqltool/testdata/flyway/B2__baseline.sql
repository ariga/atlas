CREATE TABLE post
(
    id    int NOT NULL,
    title text,
    body  text,
    created_at TIMESTAMP NOT NULL
    PRIMARY KEY (id)
);

INSERT INTO post (title, created_at) VALUES (
'This is
my multiline

value', NOW());