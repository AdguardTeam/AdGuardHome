import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactModal from 'react-modal';
import classnames from 'classnames';
import { Trans, withNamespaces } from 'react-i18next';
import { R_URL_REQUIRES_PROTOCOL } from '../../helpers/constants';
import '../ui/Modal.css';

ReactModal.setAppElement('#root');

const initialState = {
    url: '',
    name: '',
    isUrlValid: false,
};

class Modal extends Component {
    state = initialState;

    isUrlValid = url => R_URL_REQUIRES_PROTOCOL.test(url);

    handleUrlChange = async (e) => {
        const { value: url } = e.currentTarget;
        if (this.isUrlValid(url)) {
            this.setState(...this.state, { url, isUrlValid: true });
        } else {
            this.setState(...this.state, { url, isUrlValid: false });
        }
    };

    handleNameChange = (e) => {
        const { value: name } = e.currentTarget;
        this.setState({ ...this.state, name });
    };

    handleNext = () => {
        this.props.addFilter(this.state.url, this.state.name);
        setTimeout(() => {
            if (this.props.isFilterAdded) {
                this.closeModal();
            }
        }, 2000);
    };

    closeModal = () => {
        this.props.toggleModal();
        this.setState({ ...this.state, ...initialState });
    }

    render() {
        const {
            isOpen,
            title,
            inputDescription,
            processingAddFilter,
        } = this.props;
        const { isUrlValid, url, name } = this.state;
        const inputUrlClass = classnames({
            'form-control mb-2': true,
            'is-invalid': url.length > 0 && !isUrlValid,
            'is-valid': url.length > 0 && isUrlValid,
        });
        const inputNameClass = classnames({
            'form-control mb-2': true,
            'is-valid': name.length > 0,
        });

        const renderBody = () => {
            if (!this.props.isFilterAdded) {
                return (
                    <React.Fragment>
                        <input type="text" className={inputNameClass} placeholder={this.props.t('enter_name_hint')} onChange={this.handleNameChange} />
                        <input type="text" className={inputUrlClass} placeholder={this.props.t('enter_url_hint')} onChange={this.handleUrlChange} />
                        {inputDescription &&
                            <div className="description">
                                {inputDescription}
                            </div>}
                    </React.Fragment>
                );
            }
            return (
                <div className="description">
                    <Trans>filter_added_successfully</Trans>
                </div>
            );
        };

        const isValidForSubmit = !(url.length > 0 && isUrlValid && name.length > 0);

        return (
            <ReactModal
                className="Modal__Bootstrap modal-dialog modal-dialog-centered"
                closeTimeoutMS={0}
                isOpen={isOpen}
                onRequestClose={this.closeModal}
            >
                <div className="modal-content">
                    <div className="modal-header">
                    <h4 className="modal-title">
                        {title}
                    </h4>
                    <button type="button" className="close" onClick={this.closeModal}>
                        <span className="sr-only">Close</span>
                    </button>
                    </div>
                    <div className="modal-body">
                        {renderBody()}
                    </div>
                    {!this.props.isFilterAdded &&
                        <div className="modal-footer">
                            <button
                                type="button"
                                className="btn btn-secondary"
                                onClick={this.closeModal}
                            >
                                <Trans>cancel_btn</Trans>
                            </button>
                            <button
                                type="button"
                                className="btn btn-success"
                                onClick={this.handleNext}
                                disabled={isValidForSubmit || processingAddFilter}
                            >
                                <Trans>add_filter_btn</Trans>
                            </button>
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
    processingAddFilter: PropTypes.bool,
    t: PropTypes.func,
};

export default withNamespaces()(Modal);
