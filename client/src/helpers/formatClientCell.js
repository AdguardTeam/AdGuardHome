import React from 'react';
import { normalizeWhois } from './helpers';
import { WHOIS_ICONS } from './constants';

const getFormattedWhois = (whois, t) => {
    const whoisInfo = normalizeWhois(whois);
    return (
        Object.keys(whoisInfo)
            .map((key) => {
                const icon = WHOIS_ICONS[key];
                return (
                    <span className="logs__whois text-muted" key={key} title={t(key)}>
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

export const formatClientCell = (row, t, isDetailed = false) => {
    const { value, original: { info } } = row;
    let whoisContainer = '';
    let nameContainer = value;

    if (info) {
        const { name, whois_info } = info;

        if (name) {
            nameContainer = isDetailed
                ? <small title={value}>{value}</small>
                : <div className="logs__text logs__text--nowrap" title={`${name} (${value})`}>
                    {name}
                    {' '}
                    <small>{`(${value})`}</small>
                </div>;
        }

        if (whois_info) {
            whoisContainer = (
                <div className="logs__text logs__text--wrap logs__text--whois">
                    {getFormattedWhois(whois_info, t)}
                </div>
            );
        }
    }

    return (
        <div className="logs__text" title={value}>
            <>
                {nameContainer}
                {whoisContainer}
            </>
        </div>
    );
};
