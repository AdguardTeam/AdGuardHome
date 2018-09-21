import React, { Component, Fragment } from 'react';
import { HashRouter, Route } from 'react-router-dom';
import PropTypes from 'prop-types';
import LoadingBar from 'react-redux-loading-bar';

import 'react-table/react-table.css';
import '../ui/Tabler.css';
import '../ui/ReactTable.css';
import './index.css';

import Header from '../../containers/Header';
import Dashboard from '../../containers/Dashboard';
import Settings from '../../containers/Settings';
import Filters from '../../containers/Filters';
import Logs from '../../containers/Logs';
import Footer from '../ui/Footer';
import Toasts from '../Toasts';
import Status from '../ui/Status';
import Update from '../ui/Update';

class App extends Component {
    componentDidMount() {
        this.props.getDnsStatus();
        this.props.getVersion();
    }

    handleStatusChange = () => {
        this.props.enableDns();
    };

    render() {
        const { dashboard } = this.props;
        const updateAvailable =
            !dashboard.processingVersions &&
            dashboard.isCoreRunning &&
            dashboard.isUpdateAvailable;

        return (
            <HashRouter hashType='noslash'>
                <Fragment>
                    {updateAvailable &&
                        <Update
                            announcement={dashboard.announcement}
                            announcementUrl={dashboard.announcementUrl}
                        />
                    }
                    <LoadingBar className="loading-bar" updateTime={1000} />
                    <Route component={Header} />
                    <div className="container container--wrap">
                        {!dashboard.processing && !dashboard.isCoreRunning &&
                            <div className="row row-cards">
                                <div className="col-lg-12">
                                    <Status handleStatusChange={this.handleStatusChange} />
                                </div>
                            </div>
                        }
                        {!dashboard.processing && dashboard.isCoreRunning &&
                            <Fragment>
                                <Route path="/" exact component={Dashboard} />
                                <Route path="/settings" component={Settings} />
                                <Route path="/filters" component={Filters} />
                                <Route path="/logs" component={Logs} />
                            </Fragment>
                        }
                    </div>
                    <Footer />
                    <Toasts />
                </Fragment>
            </HashRouter>
        );
    }
}

App.propTypes = {
    getDnsStatus: PropTypes.func,
    enableDns: PropTypes.func,
    dashboard: PropTypes.object,
    isCoreRunning: PropTypes.bool,
    error: PropTypes.string,
    getVersion: PropTypes.func,
};

export default App;
