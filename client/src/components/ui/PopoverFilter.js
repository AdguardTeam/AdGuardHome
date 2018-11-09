import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import './Popover.css';

class PopoverFilter extends Component {
    render() {
        return (
            <div className="popover-wrap">
                <div className="popover__trigger popover__trigger--filter">
                    <svg className="popover__icon popover__icon--green" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"></circle><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path><line x1="12" y1="17" x2="12" y2="17"></line></svg>
                </div>
                <div className="popover__body popover__body--filter">
                    <div className="popover__list">
                        <div className="popover__list-item popover__list-item--nowrap">
                            <Trans>rule_label</Trans>: <strong>{this.props.rule}</strong>
                        </div>
                        {this.props.filter && <div className="popover__list-item popover__list-item--nowrap">
                            <Trans>filter_label</Trans>: <strong>{this.props.filter}</strong>
                        </div>}
                    </div>
                </div>
            </div>
        );
    }
}

PopoverFilter.propTypes = {
    rule: PropTypes.string.isRequired,
    filter: PropTypes.string,
};

export default withNamespaces()(PopoverFilter);
