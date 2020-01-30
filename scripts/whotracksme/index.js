const fs = require('fs');
const sqlite3 = require('sqlite3').verbose();
const axios = require('axios');

const INPUT_SQL_URL = 'https://raw.githubusercontent.com/cliqz-oss/whotracks.me/master/whotracksme/data/assets/trackerdb.sql';
const OUTPUT_PATH = 'whotracksme.json';

async function runScript() {
    console.log('Downloading ' + INPUT_SQL_URL);
    let response = await axios.get(INPUT_SQL_URL);
    let trackersDbSql = response.data;

    let transformToSqlite = function (sql) {
        sql = sql.trim();

        if (sql.indexOf("CREATE TABLE") >= 0) {
            sql = sql.replace(/UNIQUE/g, '');
        }

        return sql;
    }

    let whotracksme = {
        timeUpdated: new Date().toISOString(),
        categories: {},
        trackers: {},
        trackerDomains: {}
    };

    console.log('Initializing the in-memory trackers database');
    let db = new sqlite3.Database(':memory:');
    db.serialize(function () {
        trackersDbSql.split(/;\s*$/gm).forEach(function (sql) {
            sql = transformToSqlite(sql);
            db.run(sql, function () { });
        });

        db.each("SELECT * FROM categories", function (err, row) {
            if (err) {
                console.error(err);
                return;
            }

            whotracksme.categories[row.id] = row.name;
        });

        const companies = {};
        db.each("SELECT * FROM companies", function (err, row) {
            if (err) {
                console.error(err);
                return;
            }

            companies[row.id] = {
                "id": row.id,
                "name": row.name,
                "website_url": row.website_url
            };
        });

        db.each("SELECT * FROM trackers", function (err, row) {
            if (err) {
                console.error(err);
                return;
            }

            const company = companies[row.company_id];
            let url = row.website_url;
            if (!url && company) {
                url = company.website_url;
            }

            whotracksme.trackers[row.id] = {
                "name": row.name,
                "categoryId": row.category_id,
                "url": url,
            };
        });

        db.each("SELECT * FROM tracker_domains", function (err, row) {
            if (err) {
                console.error(err);
                return;
            }

            whotracksme.trackerDomains[row.domain] = row.tracker;
        });
    });

    db.close(function (err) {
        if (err) {
            console.error(err);
            return;
        }

        fs.writeFileSync(OUTPUT_PATH, JSON.stringify(whotracksme, 0, 4));
        console.log('Trackers json file has been updated: ' + OUTPUT_PATH);
    });
}

runScript();