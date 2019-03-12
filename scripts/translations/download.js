const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const requestPromise = require('request-promise');

const LOCALES_DIR = '../../client/src/__locales';
const LOCALES_LIST = [
    'en',
    'ru',
    'vi',
    'es',
    'fr',
    'ja',
    'sv',
    'pt-br',
    'zh-tw',
    'bg',
    'zh-cn',
];

/**
 * Hash content
 * @param {string} content
 */
const hashString = content => crypto.createHash('md5').update(content, 'utf8').digest('hex');

/**
 * Prepare params to get translations from oneskyapp
 * @param {string} locale language shortcut
 * @param {object} oneskyapp config oneskyapp
 */
const prepare = (locale, oneskyapp) => {
    const timestamp = Math.round(new Date().getTime() / 1000);

    let url = [];
    url.push(oneskyapp.url + oneskyapp.projectId);
    url.push(`/translations?locale=${locale}`);
    url.push('&source_file_name=en.json');
    url.push(`&export_file_name=${locale}.json`);
    url.push(`&api_key=${oneskyapp.apiKey}`);
    url.push(`&timestamp=${timestamp}`);
    url.push(`&dev_hash=${hashString(timestamp + oneskyapp.secretKey)}`);
    url = url.join('');

    return url;
};

/**
 * Promise wrapper for writing in file
 * @param {string} filename
 * @param {any} body
 */
function writeInFile(filename, body) {
    return new Promise((resolve, reject) => {
        if (typeof body !== 'string') {
            try {
                body = JSON.stringify(body, null, 4); // eslint-disable-line
            } catch (err) {
                reject(err);
            }
        }

        fs.writeFile(filename, body, (err) => {
            if (err) reject(err);
            resolve('Ok');
        });
    });
}

/**
 * Request to server onesky
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
 * Download locales
 */
const download = () => {
    const locales = LOCALES_LIST;
    let oneskyapp;
    try {
        oneskyapp = JSON.parse(fs.readFileSync('./oneskyapp.json'));
    } catch (err) {
        throw new Error(err);
    }

    const requests = locales.map((locale) => {
        const url = prepare(locale, oneskyapp);
        return request(url, locale);
    });

    Promise
        .all(requests)
        .then((res) => {
            res.forEach(item => console.log(item));
        })
        .catch(err => console.log(err));
};

download();
