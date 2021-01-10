import whotracksmeDb from './whotracksme.json';
import adguardDb from './adguard.json';
import { REPOSITORY } from '../constants';

/**
 @typedef TrackerData
 @type {object}
 @property {string} id - tracker ID.
 @property {string} name - tracker name.
 @property {string} url - tracker website url.
 @property {number} category - tracker category.
 @property {source} source - tracker data source.
 */

/**
 * Tracker data sources
 */
export const sources = {
    WHOTRACKSME: 1,
    ADGUARD: 2,
};

/**
 * Gets the source metadata for the specified tracker
 * @param {TrackerData} trackerData tracker data
 * @returns {source} source metadata or null if no matching tracker found
 */
export const getSourceData = (trackerData) => {
    if (!trackerData || !trackerData.source) {
        return null;
    }

    if (trackerData.source === sources.WHOTRACKSME) {
        return {
            name: 'Whotracks.me',
            url: `https://whotracks.me/trackers/${trackerData.id}.html`,
        };
    }
    if (trackerData.source === sources.ADGUARD) {
        return {
            name: 'AdGuard',
            url: REPOSITORY.TRACKERS_DB,
        };
    }

    return null;
};

/**
 * Gets tracker data in the specified database
 *
 * @param {String} domainName domain name to check
 * @param {*} trackersDb trackers database
 * @param {number} source source ID
 * @returns {TrackerData} tracker data or null if no matching tracker found
 */
const getTrackerDataFromDb = (domainName, trackersDb, source) => {
    if (!domainName) {
        return null;
    }

    const parts = domainName.split(/\./g)
        .reverse();
    let hostToCheck = '';

    // Check every subdomain
    for (let i = 0; i < parts.length; i += 1) {
        hostToCheck = parts[i] + (i > 0 ? '.' : '') + hostToCheck;
        const trackerId = trackersDb.trackerDomains[hostToCheck];

        if (trackerId) {
            const trackerData = trackersDb.trackers[trackerId];
            const categoryName = trackersDb.categories[trackerData.categoryId];
            trackerData.source = source;
            const sourceData = getSourceData(trackerData);

            return {
                id: trackerId,
                name: trackerData.name,
                url: trackerData.url,
                category: categoryName,
                source,
                sourceData,
            };
        }
    }

    // No tracker found for the specified domain
    return null;
};

/**
 * Gets tracker data from the trackers database
 *
 * @param {String} domainName domain name to check
 * @returns {TrackerData} tracker data or null if no matching tracker found
 */
export const getTrackerData = (domainName) => {
    if (!domainName) {
        return null;
    }

    let data = getTrackerDataFromDb(domainName, adguardDb, sources.ADGUARD);
    if (!data) {
        data = getTrackerDataFromDb(domainName, whotracksmeDb, sources.WHOTRACKSME);
    }

    return data;
};
