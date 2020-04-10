# 统计SQL的95线
SELECT
    left(any(normalized_query),20),
    count(*) AS c,
    round(avg(query_time),4) AS latency,
    round(quantile(0.95)(query_time),4) AS latency_p95,
    round((latency * c) / (max(_time) -min(_time)),4) AS load
FROM mysql_slow_log
WHERE _time >= '2020-03-26 14:12:00'
GROUP BY normalized_query
HAVING c > 1
ORDER BY c DESC
LIMIT 10;

#
# 统计全量SQL
#
SELECT
    left(replaceRegexpOne(normalized_query, '[\,|\?| ]{6,}', ',?,?,?,?,?,?') ,100),
    count(*) AS c,
    max(query_time),
    round(avg(query_time), 4) AS latency_avg,
    round(quantile(0.95)(query_time), 4) AS latency_p95,
    max(rows_examined),
    round((latency_avg * c) / (max(_time) - min(_time)), 4) AS load
FROM mysql_slow_log
WHERE normalized_query like '% tbname %'
GROUP BY replaceRegexpOne(normalized_query, '[\,|\?| ]{6,}', ',?,?,?,?,?,?')
ORDER BY replaceRegexpOne(normalized_query, '[\,|\?| ]{6,}', ',?,?,?,?,?,?') DESC
LIMIT 1000;
