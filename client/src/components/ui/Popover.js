import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';
import { getSourceData } from '../../helpers/trackers/trackers';
import { captitalizeWords } from '../../helpers/helpers';

import './Popover.css';

class Popover extends Component {
    render() {
        const { data } = this.props;

        const sourceData = getSourceData(data);

        const source = (
            <div className="popover__list-item">
                <Trans>source_label</Trans>: <a className="popover__link" target="_blank" rel="noopener noreferrer" href={sourceData.url}><strong>{sourceData.name}</strong></a>
            </div>
        );

        const tracker = (
            <div className="popover__list-item">
                <Trans>name_table_header</Trans>: <a className="popover__link" target="_blank" rel="noopener noreferrer" href={data.url}><strong>{data.name}</strong></a>
            </div>
        );

        const categoryName = captitalizeWords(data.category);

        return (
            <div className="popover-wrap">
                <div className="popover__trigger">
                    <svg className="popover__icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>
                </div>
                <div className="popover__body">
                    <div className="popover__list">
                        <div className="popover__list-title">
                            <Trans>found_in_known_domain_db</Trans>
                        </div>
                        {tracker}
                        <div className="popover__list-item">
                            <Trans>category_label</Trans>: <strong>
                            <Trans>{categoryName}</Trans></strong>
                        </div>
                        {source}
                    </div>
                </div>
            </div>
        );
    }
}

Popover.propTypes = {
    data: PropTypes.object.isRequired,
};

export default withNamespaces()(Popover);
