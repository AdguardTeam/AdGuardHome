import React, { Component } from 'react';
import PropTypes from 'prop-types';

import './Popover.css';

class Popover extends Component {
    render() {
        const { data } = this.props;

        return (
            <div className="popover-wrap">
                <div className="popover__trigger">
                    <svg className="popover__icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>
                </div>
                <div className="popover__body">
                    <div className="popover__list">
                        <div className="popover__list-title">
                            This domain belongs to a known tracker.
                        </div>
                        <div className="popover__list-item">
                            Name: <strong>{data.name}</strong>
                        </div>
                        <div className="popover__list-item">
                            Category: <strong>{data.category}</strong>
                        </div>
                        <div className="popover__list-item">
                            <a href={`https://whotracks.me/trackers/${data.id}.html`} className="popover__link" target="_blank" rel="noopener noreferrer">More information on Whotracksme</a>
                        </div>
                    </div>
                </div>
            </div>
        );
    }
}

Popover.propTypes = {
    data: PropTypes.object.isRequired,
};

export default Popover;
