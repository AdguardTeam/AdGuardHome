import React, { Fragment } from 'react';
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
                        <Fragment>
                            <svg className="logs__whois-icon icons">
                                <use xlinkHref={`#${icon}`} />
                            </svg>
                            &nbsp;
                        </Fragment>
                    )}{whoisInfo[key]}
                </span>
                );
            })
    );
};

export const formatClientCell = (row, t, isDetailed = false) => {
    const { info, client } = row.original;
    let whoisContainer = '';
    let nameContainer = client;

    if (info) {
        const { name, whois_info } = info;

        if (name) {
            nameContainer = isDetailed ? <small title={client}>{client}</small>
                : <div className="logs__text logs__text--nowrap"
                       title={`${name} (${client})`}>
                    {name}
                    <small>{`(${client})`}</small>
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
        <div className="logs__text" title={client}>
            <>
                {nameContainer}
                {whoisContainer}
            </>
        </div>
    );
};
