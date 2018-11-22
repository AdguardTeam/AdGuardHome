const path = require('path');
const fs = require('fs');
const crypto = require('crypto');
const request = require('request-promise');

const LOCALES_DIR = '../../client/src/__locales';

/**
 * Hash content
 *
 * @param {string} content
 */
const hashString = content => crypto.createHash('md5').update(content, 'utf8').digest('hex');

/**
 * Prepare post params
 */
const prepare = () => {
    let oneskyapp;
    try {
        oneskyapp = JSON.parse(fs.readFileSync('./oneskyapp.json'));
    } catch (err) {
        throw new Error(err);
    }

    const url = `${oneskyapp.url}${oneskyapp.projectId}/files`;
    const timestamp = Math.round(new Date().getTime() / 1000);
    const formData = {
        timestamp,
        file: fs.createReadStream(path.resolve(LOCALES_DIR, 'en.json')),
        file_format: 'HIERARCHICAL_JSON',
        locale: 'en',
        is_keeping_all_strings: 'false',
        api_key: oneskyapp.apiKey,
        dev_hash: hashString(timestamp + oneskyapp.secretKey),
    };

    return { url, formData };
};

/**
 * Make request to onesky to upload new json
 */
const upload = () => {
    const { url, formData } = prepare();
    request
        .post({ url, formData })
        .catch(err => console.log(err));
};

upload();
