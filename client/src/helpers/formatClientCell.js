import React from 'react';
import { normalizeWhois } from './helpers';
import { WHOIS_ICONS } from './constants';

const getFormattedWhois = (whois) => {
    const whoisInfo = normalizeWhois(whois);
    return (
        Object.keys(whoisInfo)
            .map((key) => {
                const icon = WHOIS_ICONS[key];
                return (
                    <span className="logs__whois text-muted " key={key} title={whoisInfo[key]}>
                    {icon && (
                        <>
                            <svg className="logs__whois-icon icons">
                                <use xlinkHref={`#${icon}`} />
                            </svg>
                            &nbsp;
                        </>
                    )}{whoisInfo[key]}
                </span>
                );
            })
    );
};

export const formatClientCell = (row, isDetailed = false, isLogs = true) => {
    const { value, original: { info } } = row;
    let whoisContainer = '';
    let nameContainer = value;

    if (info) {
        const { name, whois_info } = info;

        if (name) {
            if (isLogs) {
                nameContainer = !whois_info && isDetailed
                    ? (
                        <small title={value}>{value}</small>
                    ) : (
                        <div className="logs__text logs__text--nowrap" title={`${name} (${value})`}>
                            {name}&nbsp;<small>{`(${value})`}</small>
                        </div>
                    );
            } else {
                nameContainer = (
                    <div
                        className="logs__text logs__text--nowrap"
                        title={`${name} (${value})`}
                    >
                        {name}&nbsp;<small>{`(${value})`}</small>
                    </div>
                );
            }
        }

        if (whois_info && isDetailed) {
            whoisContainer = (
                <div className="logs__text logs__text--wrap logs__text--whois">
                    {getFormattedWhois(whois_info)}
                </div>
            );
        }
    }

    return (
        <div className="logs__text mw-100" title={value}>
            <>
                {nameContainer}
                {whoisContainer}
            </>
        </div>
    );
};
