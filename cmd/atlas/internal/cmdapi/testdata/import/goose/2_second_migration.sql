


-- +goose Up


ALTER TABLE post ADD updated_at TIMESTAMP NOT NULL;

-- +goose StatementBegin
-- Comment for the function declaration.
CREATE
OR REPLACE FUNCTION histories_partition_creation( DATE, DATE )
returns void AS $$
DECLARE
create_query text;
BEGIN
FOR create_query IN
SELECT 'CREATE TABLE IF NOT EXISTS histories_'
           || TO_CHAR(d, 'YYYY_MM')
           || ' ( CHECK( created_at >= timestamp '''
           || TO_CHAR(d, 'YYYY-MM-DD 00:00:00')
           || ''' AND created_at < timestamp '''
           || TO_CHAR(d + INTERVAL '1 month', 'YYYY-MM-DD 00:00:00')
           || ''' ) ) inherits ( histories );'
FROM generate_series($1, $2, '1 month') AS d LOOP
    EXECUTE create_query;
END LOOP;  -- LOOP END
END;         -- FUNCTION END
$$
language plpgsql;
-- +goose StatementEnd