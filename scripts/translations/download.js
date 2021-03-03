const fs = require('fs');
const path = require('path');
const requestPromise = require('request-promise');
const twoskyConfig = require('../../.twosky.json')[0];

const { project_id: TWOSKY_PROJECT_ID, languages } = twoskyConfig;
const LOCALES_DIR = '../../client/src/__locales';
const LOCALES_LIST = Object.keys(languages);
const BASE_FILE = 'en.json';
const TWOSKY_URI = process.env.TWOSKY_URI;

/**
 * Prepare params to get translations from twosky
 * @param {string} locale language shortcut
 * @param {object} twosky config twosky
 */
const getRequestUrl = (locale, url, projectId) => {
    return `${url}/download?format=json&language=${locale}&filename=${BASE_FILE}&project=${projectId}`;
};

/**
 * Promise wrapper for writing in file
 * @param {string} filename
 * @param {any} body
 */
function writeInFile(filename, body) {
    let normalizedBody = removeEmpty(JSON.parse(body));

    return new Promise((resolve, reject) => {
        if (typeof normalizedBody !== 'string') {
            try {
                normalizedBody = JSON.stringify(normalizedBody, null, 4); // eslint-disable-line
            } catch (err) {
                reject(err);
            }
        }

        fs.writeFile(filename, normalizedBody, (err) => {
            if (err) reject(err);
            resolve('Ok');
        });
    });
}

/**
 * Clear initial from empty value keys
 * @param {object} initialObject
 */
function removeEmpty(initialObject) {
    let processedObject = {};
    Object.keys(initialObject).forEach(prop => {
        if (initialObject[prop]) {
            processedObject[prop] = initialObject[prop];
        }
    });
    return processedObject;
}

/**
 * Request twosky
 * @param {string} url
 * @param {string} locale
 */
const request = (url, locale) => (
    requestPromise.get(url)
        .then((res) => {
            if (res.length) {
                const pathToFile = path.join(LOCALES_DIR, `${locale}.json`);
                return writeInFile(pathToFile, res);
            }
            return null;
        })
        .then((res) => {
            let result = locale;
            result += res ? ' - OK' : ' - Empty';
            return result;
        })
        .catch((err) => {
            console.log(err);
            return `${locale} - Not OK`;
        }));

/**
 * Sleep.
 * @param {number} ms
 */
const sleep = (ms) => new Promise((resolve) => {
    setTimeout(resolve, ms);
});

/**
 * Download locales
 */
const download = async () => {
    const locales = LOCALES_LIST;

    if (!TWOSKY_URI) {
        console.error('No credentials');
        return;
    }

    const requests = [];
    for (let i = 0; i < locales.length; i++) {
        const locale = locales[i];
        const url = getRequestUrl(locale, TWOSKY_URI, TWOSKY_PROJECT_ID);
        requests.push(request(url, locale));

        // Don't request the Crowdin API too aggressively to prevent spurious
        // 400 errors.
        await sleep(200);
    }

    Promise
        .all(requests)
        .then((res) => {
            res.forEach(item => console.log(item));
        })
        .catch(err => console.log(err));
};

download();
