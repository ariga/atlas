-- Comment 1.
CREATE INDEX "i" ON "s"."t" (((c #>> '{a,b,c}'::text[])));

/*
 Comment 2.
 */
SELECT name
FROM company.employees
WHERE info #>> '{department, name}' = 'Engineering';

/*
 SELECT name
FROM company.employees
WHERE info #>> '{contact, email}' = 'alice@company.com';
 */
SELECT name
FROM company.employees
WHERE info #>> '{contact, email}' = 'alice@company.com';

-- Comment 3.
CREATE INDEX "idx_emp_department" ON "company"."employees" (
    (info #>> '{department, name}')
);

/*

CREATE INDEX "idx_emp_contact" ON "company"."employees" (
    LOWER(info #>> '{contact, email}')
);
 */
CREATE INDEX "idx_emp_contact" ON "company"."employees" (
    LOWER(info #>> '{contact, email}')
);

/**
  SELECT
    info #>> '{department, name}' AS department,
    COUNT(*) AS emp_count
FROM company.employees
GROUP BY info #>> '{department, name}';
 */
SELECT
    info #>> '{department, name}' AS department,
    COUNT(*) AS emp_count
FROM company.employees
GROUP BY info #>> '{department, name}';

/**
  SELECT
    info #>> '{department, name}' AS department,
    COUNT(*) AS emp_count
FROM company.employees
GROUP BY info #>> '{department, name}';
 */
/**
  SELECT
    info #>> '{department, name}' AS department,
    COUNT(*) AS emp_count
FROM company.employees
GROUP BY info #>> '{department, name}';
 */
SELECT name
FROM company.employees
WHERE (info #>> '{department, name}' = 'Engineering'
       OR info #>> '{department, location}' = 'Building A')
  AND info #>> '{contact, email}' LIKE '%@company.com';