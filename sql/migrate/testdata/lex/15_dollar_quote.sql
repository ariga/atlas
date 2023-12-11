create function tcl_date_week(int4,int4,int4) returns text as $$
    return [clock format [clock scan "$2/$3/$1"] -format "%U"]
$$ language pltcl immutable;
create function tclsnitch() returns event_trigger language pltcl as $$
  elog NOTICE "tclsnitch: $TG_event $TG_tag"
$$;

create function foobar() returns int language sql as $$select 1;$$;

create function tcl_composite_arg_ref2(T_comp1) returns text as '
    return $1(ref2)
' language pltcl;

create function tcl_error_handling_test(text) returns text
language pltcl
as $function$
    if {[catch $1 err]} {
        if {[lindex $::errorCode 0] != "POSTGRES"} {
            return $err
        }
        array set myArray $::errorCode
        unset myArray(POSTGRES)
        unset -nocomplain myArray(funcname)
        unset -nocomplain myArray(filename)
        unset -nocomplain myArray(lineno)

# A comment.
set vals []
    foreach {key} [lsort [array names myArray]] {
set value [string map {"\n" "\n\t"} $myArray($key)]
            lappend vals "$key: $value"
        }
        return [join $vals "\n"]
    } else {
        return "no error"
    }
$function$;

create function tcl_spi_exec(
    prepare boolean,
    action text
)
    returns void language pltcl AS $function$
set query "select * from (values (1,'foo'),(2,'bar'),(3,'baz')) v(col1,col2)"
    if {$1 == "t"} {
set prep [spi_prepare $query {}]
    spi_execp -array A $prep {
        elog NOTICE "col1 $A(col1), col2 $A(col2)"

        switch $A(col1) {
            2 {
                elog NOTICE "action: $2"
                switch $2 {
                    break {
                        break
                    }
                    continue {
                        continue
                    }
                    return {
                        return
                    }
                    error {
                        error "error message"
                    }
                }
                error "should not get here"
            }
        }
    }
} else {
    spi_exec -array A $query {
        elog NOTICE "col1 $A(col1), col2 $A(col2)"

        switch $A(col1) {
            2 {
                elog NOTICE "action: $2"
                switch $2 {
                    break {
                        break
                    }
                    continue {
                        continue
                    }
                    return {
                        return
                    }
                    error {
                        error "error message"
                    }
                }
                error "should not get here"
            }
        }
    }
}
elog NOTICE "end of function"
$function$;

DO $$ BEGIN
CREATE TYPE some-type AS ENUM ('some-val-1', 'some-val-2');
EXCEPTION
    WHEN duplicate_object THEN null;
END; $$;

DO $$DECLARE r record;
BEGIN
    FOR r IN SELECT table_schema, table_name FROM information_schema.tables
             WHERE table_type = 'VIEW' AND table_schema = 'public'
    LOOP
        EXECUTE 'GRANT ALL ON ' || quote_ident(r.table_schema) || '.' || quote_ident(r.table_name) || ' TO webuser';
    END LOOP;
END$$;

# Shorter examples.
SELECT * FROM table WHERE name = $$John Doe$$;
SELECT * FROM table WHERE name = $a1$John Doe$a1$;
UPDATE table SET description = $$Lorem ipsum dolor sit amet$$ WHERE id = 123;
INSERT INTO table (name, description) VALUES ($$Jane Smith$$, $$This is Jane's description$$);
CREATE INDEX index ON table (c);
CREATE TRIGGER trigger BEFORE INSERT ON table FOR EACH ROW EXECUTE FUNCTION function(params);
SELECT COUNT(*) FROM table WHERE description ILIKE $$%foo%$$;
CREATE FUNCTION my_function() RETURNS void AS $$
BEGIN
   -- function body here
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION f1(target regclass)
  RETURNS VOID AS $BEGIN$
DECLARE
table_name TEXT := quote_ident(target :: text);
BEGIN
EXECUTE 'alter table ' || table_name || ' add column id serial';
END;
$BEGIN$
LANGUAGE plpgsql;