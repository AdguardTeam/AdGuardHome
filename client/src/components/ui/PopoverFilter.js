import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import './Popover.css';

class PopoverFilter extends Component {
    render() {
        const { rule, filter, service } = this.props;

        if (!rule && !service) {
            return '';
        }

        return (
            <div className="popover-wrap">
                <div className="popover__trigger popover__trigger--filter">
                    <svg className="popover__icon popover__icon--green">
                        <use xlinkHref="#question" />
                    </svg>
                </div>
                <div className="popover__body popover__body--filter">
                    <div className="popover__list">
                        {rule && (
                            <div className="popover__list-item popover__list-item--nowrap">
                                <Trans>rule_label</Trans>: <strong>{rule}</strong>
                            </div>
                        )}
                        {filter && (
                            <div className="popover__list-item popover__list-item--nowrap">
                                <Trans>list_label</Trans>: <strong>{filter}</strong>
                            </div>
                        )}
                        {service && (
                            <div className="popover__list-item popover__list-item--nowrap">
                                <Trans>blocked_service</Trans>: <strong>{service}</strong>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        );
    }
}

PopoverFilter.propTypes = {
    rule: PropTypes.string,
    filter: PropTypes.string,
    service: PropTypes.string,
};

export default withNamespaces()(PopoverFilter);
