import whotracksmeWebsites from './whotracksme_web.json';

import trackersDb from './trackers.json';
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
 * Gets link to tracker page on whotracks.me.
 *
 * @param trackerId
 * @return {string}
 */
const getWhotracksmeUrl = (trackerId: any) => {
    const websiteId = whotracksmeWebsites.websites[trackerId];
    if (websiteId) {
        // Overrides links to websites.
        return `https://whotracks.me/websites/${websiteId}.html`;
    }

    return `https://whotracks.me/trackers/${trackerId}.html`;
};

/**
 * Gets the source metadata for the specified tracker
 *
 * @param {TrackerData} trackerData tracker data
 * @returns {source} source metadata or null if no matching tracker found
 */
export const getSourceData = (trackerData: any) => {
    if (!trackerData || !trackerData.source) {
        return null;
    }

    if (trackerData.source === sources.WHOTRACKSME) {
        return {
            name: 'Whotracks.me',
            url: getWhotracksmeUrl(trackerData.id),
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
 * Converts the JSON string source into numeric source for AdGuard Home
 *
 * @param {TrackerData} trackerData tracker data
 * @returns {number} source number
 */
const convertSource = (sourceStr: any) => {
    if (!sourceStr || sourceStr !== 'AdGuard') {
        return sources.WHOTRACKSME;
    }

    return sources.ADGUARD;
};

/**
 * Gets tracker data from the trackers database
 *
 * @param {String} domainName domain name to check
 * @returns {TrackerData} tracker data or null if no matching tracker found
 */
export const getTrackerData = (domainName: any) => {
    if (!domainName) {
        return null;
    }

    const parts = domainName.split(/\./g).reverse();
    let hostToCheck = '';

    // Check every subdomain
    for (let i = 0; i < parts.length; i += 1) {
        hostToCheck = parts[i] + (i > 0 ? '.' : '') + hostToCheck;
        const trackerId = trackersDb.trackerDomains[hostToCheck];

        if (trackerId) {
            const trackerData = trackersDb.trackers[trackerId];
            const categoryName = trackersDb.categories[trackerData.categoryId];
            const source = convertSource(trackerData.source);
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
