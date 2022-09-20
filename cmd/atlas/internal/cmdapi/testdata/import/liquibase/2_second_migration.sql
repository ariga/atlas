--liquibase formatted sql

--changeset atlas:2-1
CREATE TABLE tbl_2 (col INT);
--rollback DROP TABLE tbl_2;
