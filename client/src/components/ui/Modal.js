import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactModal from 'react-modal';
import classnames from 'classnames';
import { R_URL_REQUIRES_PROTOCOL } from '../../helpers/constants';
import './Modal.css';

ReactModal.setAppElement('#root');

export default class Modal extends Component {
    state = {
        url: '',
        isUrlValid: false,
    };

    // eslint-disable-next-line
    isUrlValid = url => {
        return R_URL_REQUIRES_PROTOCOL.test(url);
    };

    handleUrlChange = async (e) => {
        const { value: url } = e.currentTarget;
        if (this.isUrlValid(url)) {
            this.setState(...this.state, { url, isUrlValid: true });
        } else {
            this.setState(...this.state, { url, isUrlValid: false });
        }
    };

    handleNext = () => {
        this.props.addFilter(this.state.url);
        setTimeout(() => {
            if (this.props.isFilterAdded) {
                this.props.toggleModal();
            }
        }, 2000);
    };
    render() {
        const {
            isOpen,
            toggleModal,
            title,
            inputDescription,
        } = this.props;
        const { isUrlValid, url } = this.state;
        const inputClass = classnames({
            'form-control mb-2': true,
            'is-invalid': url.length > 0 && !isUrlValid,
            'is-valid': url.length > 0 && isUrlValid,
        });

        const renderBody = () => {
            if (!this.props.isFilterAdded) {
                return (
                    <React.Fragment>
                        <input type="text" className={inputClass} placeholder="Enter URL or path" onChange={this.handleUrlChange}/>
                        {inputDescription &&
                            <div className="description">
                                {inputDescription}
                            </div>}
                    </React.Fragment>
                );
            }
            return (
                <div className="description">
                    Url added successfully
                </div>
            );
        };

        const isValidForSubmit = !(url.length > 0 && isUrlValid);

        return (
            <ReactModal
                className="Modal__Bootstrap modal-dialog modal-dialog-centered"
                closeTimeoutMS={0}
                isOpen={ isOpen }
                onRequestClose={toggleModal}
            >
                <div className="modal-content">
                    <div className="modal-header">
                    <h4 className="modal-title">
                        {title}
                    </h4>
                    <button type="button" className="close" onClick={toggleModal}>
                        <span className="sr-only">Close</span>
                    </button>
                    </div>
                    <div className="modal-body">
                        { renderBody()}
                    </div>
                    {
                        !this.props.isFilterAdded &&
                            <div className="modal-footer">
                                <button type="button" className="btn btn-secondary" onClick={toggleModal}>Cancel</button>
                                <button type="button" className="btn btn-success" onClick={this.handleNext} disabled={isValidForSubmit}>Add filter</button>
                            </div>
                    }
                </div>
            </ReactModal>
        );
    }
}

Modal.propTypes = {
    toggleModal: PropTypes.func.isRequired,
    isOpen: PropTypes.bool.isRequired,
    title: PropTypes.string.isRequired,
    inputDescription: PropTypes.string,
    addFilter: PropTypes.func.isRequired,
    isFilterAdded: PropTypes.bool,
};
