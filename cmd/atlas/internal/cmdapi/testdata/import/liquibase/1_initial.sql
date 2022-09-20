--liquibase formatted sql

--changeset atlas:1-1
CREATE TABLE post
(
    id    int NOT NULL,
    title text,
    body  text,
    PRIMARY KEY (id)
);
--rollback: DROP TABLE post;

--changeset atlas:1-2
ALTER TABLE post ADD created_at TIMESTAMP NOT NULL;
--rollback: ALTER TABLE post DROP created_at;

--changeset atlas:1-3
INSERT INTO post (title) VALUES (
'This is
my multiline

value');
