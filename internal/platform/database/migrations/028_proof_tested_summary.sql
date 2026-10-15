-- The "How it was tested" line of a proof used to list every command that ran, which is long. It now says how many
-- checks ran and how they ended. Tasks that finished before that get the same wording, counted from the checks
-- already stored in their proof (each verified check is one run the server made).
UPDATE tasks
   SET proof = json_set(proof, '$.rows[1].value', (
         SELECT CASE
                  WHEN total = 1 AND passed = 1 THEN 'Ran 1 check, and it passed'
                  WHEN total = 1 THEN 'Ran 1 check, and it failed'
                  WHEN passed = total THEN 'Ran ' || total || ' checks, all passed'
                  WHEN passed = 0 THEN 'Ran ' || total || ' checks, all failed'
                  ELSE 'Ran ' || total || ' checks: ' || passed || ' passed, ' || (total - passed) || ' failed'
                END
           FROM (SELECT count(*) AS total,
                        coalesce(sum(json_extract(c.value, '$.passed')), 0) AS passed
                   FROM json_each(tasks.proof, '$.checks') AS c
                  WHERE json_extract(c.value, '$.verified') = 1)))
 WHERE proof IS NOT NULL
   AND json_valid(proof)
   AND json_extract(proof, '$.rows[1].label') = 'How it was tested'
   AND json_extract(proof, '$.rows[1].value') LIKE 'Ran %'
   AND (SELECT count(*) FROM json_each(tasks.proof, '$.checks') AS c WHERE json_extract(c.value, '$.verified') = 1) > 0;
