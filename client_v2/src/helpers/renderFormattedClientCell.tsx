import { Show } from 'solid-js';
import type { JSXElement } from 'solid-js';

import { A } from '@solidjs/router';

import { normalizeWhois } from './helpers';
import { WHOIS_ICONS } from './constants';
import type { QueryLogItemClientWhois } from 'panel/api/model/queryLogItemClientWhois';

type ClientCellInfo = {
    name?: string;
    whois_info?: QueryLogItemClientWhois;
};

const getFormattedWhois = (whois: QueryLogItemClientWhois) => {
    const whoisInfo = normalizeWhois(whois);
    return Object.entries(whoisInfo).map(([key, value]) => {
        const icon = WHOIS_ICONS[key as keyof typeof WHOIS_ICONS];
        const strValue = String(value ?? '');
        return (
            <span class="logs__whois text-muted" title={strValue}>
                <Show when={icon}>
                    <svg class="logs__whois-icon icons icon--18">
                        <use href={`#${icon}`} />
                    </svg>
                    &nbsp;
                </Show>
                {strValue}
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
export const renderFormattedClientCell = (
    value: string,
    info: ClientCellInfo | null,
    isDetailed = false,
    isLogs = false,
) => {
    let whoisContainer: JSXElement = null;
    let nameContainer: JSXElement = value;

    if (info) {
        const { name, whois_info } = info;
        const whoisAvailable = whois_info && Object.keys(whois_info).length > 0;

        if (name) {
            const nameValue = (
                <div
                    class="logs__text logs__text--link logs__text--nowrap logs__text--client"
                    title={`${name} (${value})`}
                >
                    {name}&nbsp;<small>{`(${value})`}</small>
                </div>
            );

            if (!isLogs) {
                nameContainer = nameValue;
            } else {
                nameContainer =
                    !whoisAvailable && isDetailed ? (
                        <small title={value}>{value}</small>
                    ) : (
                        nameValue
                    );
            }
        }

        if (whoisAvailable && isDetailed) {
            whoisContainer = (
                <div class="logs__text logs__text--wrap logs__text--whois">
                    {getFormattedWhois(whois_info)}
                </div>
            );
        }
    }

    return (
        <div class="logs__text logs__text--client mw-100" title={value}>
            <A href={`logs?search="${encodeURIComponent(value)}"`}>{nameContainer}</A>
            {whoisContainer}
        </div>
    );
};
