import React from 'react';

import { Link } from 'react-router-dom';

import { normalizeWhois } from './helpers';
import { WHOIS_ICONS } from './constants';

const getFormattedWhois = (whois: any) => {
    const whoisInfo = normalizeWhois(whois);
    return Object.keys(whoisInfo).map((key) => {
        const icon = WHOIS_ICONS[key];
        return (
            <span className="logs__whois text-muted" key={key} title={whoisInfo[key]}>
                {icon && (
                    <>
                        <svg className="logs__whois-icon icons icon--18">
                            <use xlinkHref={`#${icon}`} />
                        </svg>
                        &nbsp;
                    </>
                )}
                {whoisInfo[key]}
            </span>
        );
    });
};

/**
 * @param {string} value
 * @param {object} info
 * @param {string} info.name
 * @param {object} info.whois_info
 * @param {boolean} [isDetailed]
 * @param {boolean} [isLogs]
 * @returns {JSXElement}
 */
export const renderFormattedClientCell = (value: any, info: any, isDetailed = false, isLogs = false) => {
    let whoisContainer = null;
    let nameContainer = value;

    if (info) {
        const { name, whois_info } = info;
        const whoisAvailable = whois_info && Object.keys(whois_info).length > 0;

        if (name) {
            const nameValue = (
                <div
                    className="logs__text logs__text--link logs__text--nowrap logs__text--client"
                    title={`${name} (${value})`}>
                    {name}&nbsp;<small>{`(${value})`}</small>
                </div>
            );

            if (!isLogs) {
                nameContainer = nameValue;
            } else {
                nameContainer = !whoisAvailable && isDetailed ? <small title={value}>{value}</small> : nameValue;
            }
        }

        if (whoisAvailable && isDetailed) {
            whoisContainer = (
                <div className="logs__text logs__text--wrap logs__text--whois">{getFormattedWhois(whois_info)}</div>
            );
        }
    }

    return (
        <div className="logs__text logs__text--client mw-100" title={value}>
            <Link to={`logs?search="${encodeURIComponent(value)}"`}>{nameContainer}</Link>
            {whoisContainer}
        </div>
    );
};
