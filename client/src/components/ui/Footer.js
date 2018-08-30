import React, { Component } from 'react';

class Footer extends Component {
    getYear = () => {
        const today = new Date();
        return today.getFullYear();
    };

    render() {
        return (
            <footer className="footer">
                <div className="container">
                    <div className="row align-items-center flex-row-reverse">
                        <div className="col-12 col-lg-auto ml-lg-auto">
                            <ul className="list-inline list-inline-dots text-center mb-0">
                                <li className="list-inline-item">
                                    <a href="https://adguard.com/welcome.html" target="_blank" rel="noopener noreferrer">Homepage</a>
                                </li>
                                <li className="list-inline-item">
                                    <a href="https://github.com/AdguardTeam/" target="_blank" rel="noopener noreferrer">Github</a>
                                </li>
                                <li className="list-inline-item">
                                    <a href="https://adguard.com/privacy.html" target="_blank" rel="noopener noreferrer">Privacy Policy</a>
                                </li>
                            </ul>
                        </div>
                        <div className="col-12 col-lg-auto mt-3 mt-lg-0 text-center">
                            © AdGuard {this.getYear()}
                        </div>
                    </div>
                </div>
            </footer>
        );
    }
}

export default Footer;
