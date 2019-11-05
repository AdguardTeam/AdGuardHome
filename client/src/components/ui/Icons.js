import React from 'react';

import './Icons.css';

const Icons = () => (
    <svg xmlns="http://www.w3.org/2000/svg" className="hidden">
        <symbol id="android" viewBox="0 0 14 16" fill="currentColor">
            <path d="M11.2 5.2H2.8c-.2 0-.3.1-.3.3v6.7c0 .2.2.3.3.3h1.5v2.3c0 .5.4.9 1 .9s1-.4 1-.9v-2.3h1.4v2.3c0 .5.4.9 1 .9s1-.4 1-.9v-2.3h1.5c.2 0 .3-.1.3-.3V5.5c.1-.2-.1-.3-.3-.3zM1 5.2c-.6 0-1 .4-1 .9V10c0 .5.4.9 1 .9s1-.4 1-.9V6.1c0-.5-.4-.9-1-.9zm12 0c-.6 0-1 .4-1 .9V10c0 .5.4.9 1 .9s1-.4 1-.9V6.1c0-.5-.5-.9-1-.9zM2.9 4.7h8.3c.2 0 .4-.2.3-.4-.3-1.2-1.1-2.3-2.2-2.9L10 .3c0-.1 0-.2-.1-.2-.1-.1-.2-.1-.3 0l-.7 1.2C8.3 1 7.7.9 7 .9s-1.3.1-1.9.4L4.4.1C4.3 0 4.2 0 4.1 0c-.1.1-.1.2 0 .3l.7 1.2c-1.1.6-2 1.6-2.2 2.9-.1.1.1.3.3.3zm6.2-2.1c.2 0 .4.2.4.4s-.2.3-.4.3-.4-.2-.4-.4.2-.3.4-.3zm-4.2 0c.2 0 .4.2.4.4s-.2.4-.4.4-.4-.2-.4-.4.2-.4.4-.4z"/>
        </symbol>

        <symbol id="macos" viewBox="0 0 42 42" fill="currentColor">
            <path d="M23.091,14.018 L23.091,13.676 L22.028,13.749 C21.727,13.768 21.501,13.832 21.349,13.94 C21.197,14.049 21.121,14.2 21.121,14.393 C21.121,14.581 21.196,14.731 21.347,14.842 C21.497,14.954 21.699,15.009 21.951,15.009 C22.112,15.009 22.263,14.984 22.402,14.935 C22.541,14.886 22.663,14.817 22.765,14.729 C22.867,14.642 22.947,14.538 23.004,14.417 C23.062,14.296 23.091,14.163 23.091,14.018 Z M21,0.25 C9.421,0.25 0.25,9.421 0.25,21 C0.25,32.58 9.421,41.75 21,41.75 C32.579,41.75 41.75,32.58 41.75,21 C41.75,9.421 32.58,0.25 21,0.25 Z M25.028,12.549 C25.126,12.274 25.264,12.038 25.443,11.842 C25.622,11.646 25.837,11.495 26.089,11.389 C26.341,11.283 26.622,11.23 26.931,11.23 C27.21,11.23 27.462,11.272 27.686,11.355 C27.911,11.438 28.103,11.55 28.264,11.691 C28.425,11.832 28.553,11.996 28.647,12.184 C28.741,12.372 28.797,12.571 28.816,12.78 L27.983,12.78 C27.962,12.665 27.924,12.557 27.87,12.458 C27.816,12.359 27.745,12.273 27.657,12.2 C27.568,12.127 27.464,12.07 27.345,12.029 C27.225,11.987 27.091,11.967 26.94,11.967 C26.763,11.967 26.602,12.003 26.459,12.074 C26.315,12.145 26.192,12.246 26.09,12.376 C25.988,12.506 25.909,12.665 25.853,12.851 C25.796,13.038 25.768,13.245 25.768,13.473 C25.768,13.709 25.796,13.921 25.853,14.107 C25.909,14.294 25.989,14.451 26.093,14.58 C26.196,14.709 26.321,14.808 26.466,14.876 C26.611,14.944 26.771,14.979 26.945,14.979 C27.23,14.979 27.462,14.912 27.642,14.778 C27.822,14.644 27.938,14.448 27.992,14.19 L28.826,14.19 C28.802,14.418 28.739,14.626 28.637,14.814 C28.535,15.002 28.403,15.162 28.241,15.295 C28.078,15.428 27.887,15.531 27.667,15.603 C27.447,15.675 27.205,15.712 26.942,15.712 C26.63,15.712 26.349,15.66 26.096,15.557 C25.844,15.454 25.627,15.305 25.447,15.112 C25.267,14.919 25.128,14.684 25.03,14.407 C24.932,14.13 24.883,13.819 24.883,13.472 C24.881,13.133 24.93,12.825 25.028,12.549 Z M13.175,11.287 L14.009,11.287 L14.009,12.028 L14.025,12.028 C14.076,11.905 14.143,11.794 14.225,11.698 C14.307,11.601 14.401,11.519 14.509,11.45 C14.616,11.381 14.735,11.329 14.863,11.293 C14.992,11.257 15.128,11.239 15.27,11.239 C15.576,11.239 15.835,11.312 16.045,11.458 C16.256,11.604 16.406,11.814 16.494,12.088 L16.515,12.088 C16.571,11.956 16.645,11.838 16.736,11.734 C16.827,11.63 16.932,11.54 17.05,11.466 C17.168,11.392 17.298,11.336 17.439,11.297 C17.58,11.258 17.728,11.239 17.884,11.239 C18.099,11.239 18.294,11.273 18.47,11.342 C18.646,11.411 18.796,11.507 18.921,11.632 C19.046,11.757 19.142,11.909 19.209,12.087 C19.276,12.265 19.31,12.463 19.31,12.681 L19.31,15.662 L18.44,15.662 L18.44,12.89 C18.44,12.603 18.366,12.38 18.218,12.223 C18.071,12.066 17.86,11.987 17.586,11.987 C17.452,11.987 17.329,12.011 17.217,12.058 C17.106,12.105 17.009,12.171 16.929,12.256 C16.848,12.34 16.785,12.442 16.74,12.56 C16.694,12.678 16.671,12.807 16.671,12.947 L16.671,15.662 L15.813,15.662 L15.813,12.818 C15.813,12.692 15.793,12.578 15.754,12.476 C15.715,12.374 15.66,12.287 15.587,12.214 C15.515,12.141 15.426,12.086 15.323,12.047 C15.219,12.008 15.103,11.988 14.974,11.988 C14.84,11.988 14.716,12.013 14.601,12.063 C14.487,12.113 14.389,12.182 14.307,12.27 C14.225,12.359 14.161,12.463 14.116,12.584 C14.072,12.704 14,12.836 14,12.978 L14,15.661 L13.175,15.661 L13.175,11.287 Z M15.068,32.226 C11.243,32.226 8.844,29.568 8.844,25.326 C8.844,21.084 11.243,18.417 15.068,18.417 C18.893,18.417 21.283,21.084 21.283,25.326 C21.283,29.567 18.893,32.226 15.068,32.226 Z M22.15,15.651 C22.009,15.687 21.865,15.705 21.717,15.705 C21.499,15.705 21.3,15.674 21.119,15.612 C20.937,15.55 20.782,15.463 20.652,15.35 C20.522,15.237 20.42,15.101 20.348,14.941 C20.275,14.781 20.239,14.603 20.239,14.407 C20.239,14.023 20.382,13.723 20.668,13.507 C20.954,13.291 21.368,13.165 21.911,13.13 L23.091,13.062 L23.091,12.724 C23.091,12.472 23.011,12.279 22.851,12.148 C22.691,12.017 22.465,11.951 22.172,11.951 C22.054,11.951 21.943,11.966 21.841,11.995 C21.739,12.025 21.649,12.067 21.571,12.122 C21.493,12.177 21.428,12.243 21.378,12.32 C21.327,12.396 21.292,12.482 21.273,12.576 L20.455,12.576 C20.46,12.383 20.508,12.204 20.598,12.04 C20.688,11.876 20.81,11.734 20.965,11.613 C21.12,11.492 21.301,11.398 21.511,11.331 C21.721,11.264 21.949,11.23 22.196,11.23 C22.462,11.23 22.703,11.263 22.919,11.331 C23.135,11.399 23.32,11.494 23.473,11.619 C23.626,11.744 23.744,11.894 23.827,12.07 C23.91,12.246 23.952,12.443 23.952,12.66 L23.952,15.661 L23.119,15.661 L23.119,14.932 L23.098,14.932 C23.036,15.05 22.958,15.157 22.863,15.252 C22.767,15.347 22.66,15.429 22.541,15.496 C22.421,15.563 22.291,15.615 22.15,15.651 Z M27.653,32.226 C24.736,32.226 22.753,30.698 22.615,28.299 L24.514,28.299 C24.662,29.67 25.987,30.578 27.802,30.578 C29.543,30.578 30.794,29.67 30.794,28.429 C30.794,27.355 30.034,26.706 28.275,26.262 L26.561,25.836 C24.097,25.225 22.977,24.104 22.977,22.261 C22.977,19.992 24.959,18.417 27.784,18.417 C30.544,18.417 32.47,20.001 32.544,22.279 L30.664,22.279 C30.534,20.908 29.414,20.065 27.746,20.065 C26.088,20.065 24.94,20.917 24.94,22.149 C24.94,23.121 25.662,23.696 27.422,24.14 L28.867,24.501 C31.618,25.168 32.748,26.252 32.748,28.197 C32.747,30.679 30.784,32.226 27.653,32.226 Z M15.068,20.12 C12.447,20.12 10.808,22.13 10.808,25.325 C10.808,28.511 12.447,30.521 15.068,30.521 C17.68,30.521 19.328,28.511 19.328,25.325 C19.329,22.13 17.68,20.12 15.068,20.12 Z" />
        </symbol>

        <symbol id="windows" viewBox="0 0 14 16" fill="currentColor">
            <path d="M0 13.7L6.5 14.6 6.5 8.4 0 8.4z"/><path d="M0 7.6L6.5 7.6 6.5 1.3 0 2.2z"/><path d="M7.2 14.7L15.9 15.9 15.9 8.4 15.9 8.4 7.2 8.4z"/><path d="M7.2 1.2L7.2 7.6 15.9 7.6 15.9 0z"/>
        </symbol>

        <symbol id="ios" viewBox="0 0 512 512" fill="currentColor">
            <path d="M395.748 272.046c-.646-64.841 52.88-95.938 55.271-97.483-30.075-44.01-76.925-50.039-93.62-50.736-39.871-4.037-77.798 23.474-98.033 23.474-20.184 0-51.409-22.877-84.476-22.276-43.458.646-83.529 25.269-105.906 64.19-45.152 78.35-11.563 194.42 32.445 257.963 21.504 31.104 47.146 66.038 80.813 64.79 32.421-1.294 44.681-20.979 83.878-20.979 39.196 0 50.215 20.979 84.524 20.335 34.888-.648 56.991-31.699 78.347-62.898 24.694-36.084 34.862-71.019 35.462-72.812-.775-.354-68.031-26.119-68.705-103.568zM331.28 81.761C349.149 60.082 361.21 30.005 357.92 0c-25.739 1.048-56.938 17.145-75.405 38.775-16.57 19.188-31.075 49.813-27.188 79.218 28.734 2.242 58.065-14.602 75.953-36.232z"/>
        </symbol>

        <symbol id="router" viewBox="0 0 30 30" fill="currentColor">
            <path d="M17.646 2.332a1 1 0 0 0-.697 1.719 6.984 6.984 0 0 1 0 9.898 1 1 0 1 0 1.414 1.414c3.507-3.506 3.507-9.22 0-12.726a1 1 0 0 0-.717-.305zm-12.662.654A1 1 0 0 0 4 4v14a2 2 0 0 0-2 2v4a2 2 0 0 0 2 2h22a2 2 0 0 0 2-2v-4a2 2 0 0 0-2-2H12V9a1 1 0 0 0-1.016-1.014A1 1 0 0 0 10 9v9H6V4a1 1 0 0 0-1.016-1.014zm9.834 2.176a1 1 0 0 0-.697 1.717 2.985 2.985 0 0 1 0 4.242 1 1 0 1 0 1.414 1.414 5.014 5.014 0 0 0 0-7.07 1 1 0 0 0-.717-.303zM5 21a1 1 0 1 1 0 2 1 1 0 0 1 0-2zm4 0a1 1 0 1 1 0 2 1 1 0 0 1 0-2z" />
        </symbol>

        <symbol id="edit" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
        </symbol>

        <symbol id="delete" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="m3 6h2 16"/><path d="m19 6v14a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2-2v-14m3 0v-2a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><path d="m10 11v6"/><path d="m14 11v6"/>
        </symbol>

        <symbol id="back" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="m19 12h-14"/><path d="m12 19-7-7 7-7"/>
        </symbol>

        <symbol id="dashboard" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="m3 9 9-7 9 7v11a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2-2z"/><path d="m9 22v-10h6v10"/>
        </symbol>

        <symbol id="filters" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="m22 3h-20l8 9.46v6.54l4 2v-8.54z"/>
        </symbol>

        <symbol id="log" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="m14 2h-8a2 2 0 0 0 -2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-12z"/><path d="m14 2v6h6"/><path d="m16 13h-8"/><path d="m16 17h-8"/><path d="m10 9h-1-1"/>
        </symbol>

        <symbol id="setup" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <circle cx="12" cy="12" r="10"></circle><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path><line x1="12" y1="17" x2="12" y2="17"></line>
        </symbol>

        <symbol id="settings" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <circle cx="12" cy="12" r="3"/><path d="m19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1 -2.83 0l-.06-.06a1.65 1.65 0 0 0 -1.82-.33 1.65 1.65 0 0 0 -1 1.51v.17a2 2 0 0 1 -2 2 2 2 0 0 1 -2-2v-.09a1.65 1.65 0 0 0 -1.08-1.51 1.65 1.65 0 0 0 -1.82.33l-.06.06a2 2 0 0 1 -2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0 -1.51-1h-.17a2 2 0 0 1 -2-2 2 2 0 0 1 2-2h.09a1.65 1.65 0 0 0 1.51-1.08 1.65 1.65 0 0 0 -.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33h.08a1.65 1.65 0 0 0 1-1.51v-.17a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0 -.33 1.82v.08a1.65 1.65 0 0 0 1.51 1h.17a2 2 0 0 1 2 2 2 2 0 0 1 -2 2h-.09a1.65 1.65 0 0 0 -1.51 1z"/>
        </symbol>

        <symbol id="refresh" viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M23 4v6h-6M1 20v-6h6"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
        </symbol>

        <symbol id="dns_privacy" viewBox="0 0 30 30" stroke="none" fill="currentColor" strokeLinecap="round" strokeLinejoin="round">
            <path d="M15 3C10.57 3 6.701 5.419 4.623 9h2.39a10.063 10.063 0 0 1 4.05-3.19c-.524.89-.961 1.973-1.3 3.19h2.108c.79-2.459 1.998-4 3.129-4s2.339 1.541 3.129 4h2.107c-.338-1.217-.774-2.3-1.299-3.19A10.062 10.062 0 0 1 22.989 9h2.389C23.298 5.419 19.43 3 15 3zm7.035 9.129c-1.372 0-2.264.73-2.264 1.842 0 .896.538 1.463 1.579 1.66l.75.15c.65.13.898.3.898.615 0 .375-.37.635-.91.635-.6 0-1.014-.265-1.049-.68h-1.38c.023 1.097.93 1.776 2.37 1.776 1.491 0 2.399-.717 2.399-1.904 0-.903-.504-1.412-1.63-1.63l-.734-.142c-.6-.118-.851-.3-.851-.611 0-.378.336-.62.844-.62.509 0 .891.28.923.682h1.336c-.024-1.053-.948-1.773-2.28-1.773zm-16.185.148v5.696h2.39c1.712 0 2.662-1.033 2.662-2.903 0-1.779-.966-2.793-2.662-2.793H5.85zm6.933.004v5.692h1.373v-3.235h.076l2.377 3.235h1.149V12.28h-1.373v3.203h-.076l-2.372-3.203h-1.154zm-5.486 1.16h.682c.912 0 1.449.596 1.449 1.657 0 1.128-.51 1.713-1.45 1.713h-.681v-3.37zM4.623 21C6.701 24.581 10.57 27 15 27c4.43 0 8.299-2.419 10.377-6h-2.389a10.063 10.063 0 0 1-4.049 3.19c.524-.89.96-1.973 1.297-3.19H18.13c-.79 2.459-1.996 4-3.127 4-1.131 0-2.339-1.541-3.129-4h-2.11c.339 1.217.776 2.3 1.3 3.19A10.056 10.056 0 0 1 7.013 21h-2.39z"></path>
        </symbol>

        <symbol id="service_amazon" viewBox="0 0 32 32" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M16.2,4c-3.3,0-6.9,1.2-7.7,5.3C8.4,9.7,8.7,10,9,10l3.3,0.3c0.3,0,0.6-0.3,0.6-0.6c0.3-1.4,1.5-2.1,2.8-2.1c0.7,0,1.5,0.3,1.9,0.9c0.5,0.7,0.4,1.7,0.4,2.5v0.5c-2,0.2-4.6,0.4-6.5,1.2c-2.2,0.9-3.7,2.8-3.7,5.7c0,3.6,2.3,5.4,5.2,5.4c2.5,0,3.8-0.6,5.7-2.5c0.6,0.9,0.9,1.4,2,2.3c0.3,0.1,0.6,0.1,0.8-0.1v0c0.7-0.6,2-1.7,2.7-2.3c0.3-0.2,0.2-0.6,0-0.9c-0.6-0.9-1.3-1.6-1.3-3.2v-5.4c0-2.3,0.2-4.4-1.5-6C20.1,4.4,17.9,4,16.2,4z M17.1,14.3c0.3,0,0.6,0,0.9,0v0.8c0,1.3,0.1,2.5-0.6,3.7c-0.5,1-1.4,1.6-2.4,1.6c-1.3,0-2.1-1-2.1-2.5C12.9,15.2,14.9,14.5,17.1,14.3z M26.7,22.4c-0.9,0-1.9,0.2-2.7,0.8c-0.2,0.2-0.2,0.4,0.1,0.4c0.9-0.1,2.8-0.4,3.2,0.1s-0.4,2.3-0.7,3.1c-0.1,0.2,0.1,0.3,0.3,0.2c1.5-1.2,1.9-3.8,1.6-4.2C28.3,22.5,27.6,22.4,26.7,22.4z M3.7,22.8c-0.2,0-0.3,0.3-0.1,0.4c3.3,3,7.6,4.7,12.4,4.7c3.4,0,7.4-1.1,10.2-3.1c0.5-0.3,0.1-0.9-0.4-0.7c-3.1,1.3-6.4,1.9-9.5,1.9c-4.5,0-8.8-1.2-12.4-3.3C3.8,22.9,3.7,22.8,3.7,22.8z" />
        </symbol>

        <symbol id="service_youtube" viewBox="0 0 24 24" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M19.695 4.04S15.348 3.2 12 3.2s-7.695.84-7.695.84L1.602 7.2v9.6l2.703 3.16s4.347.84 7.695.84 7.695-.84 7.695-.84l2.703-3.16V12 7.2zM9.602 15.68V8.32L16 12zm0 0"/><path d="M19.2 4a3.198 3.198 0 1 0 0 6.398c1.769 0 3.198-1.43 3.198-3.199C22.398 5.434 20.968 4 19.2 4zm0 9.602a3.198 3.198 0 1 0 0 6.398c1.769 0 3.198-1.434 3.198-3.2 0-1.769-1.43-3.198-3.199-3.198zM1.601 7.199c0 1.77 1.43 3.2 3.199 3.2 1.765 0 2.398-1.43 2.398-3.2C7.2 5.434 6.566 4 4.801 4 3.03 4 1.6 5.434 1.6 7.2zM4.8 13.602c-1.77 0-3.2 1.43-3.2 3.199A3.198 3.198 0 1 0 8 16.8c0-1.77-1.434-3.2-3.2-3.2zm0 0" />
        </symbol>

        <symbol id="service_discord" viewBox="0 0 24 24" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M20.098 5.559C18.156 4 15.09 3.734 14.96 3.723a.493.493 0 0 0-.484.285c-.004.008-.172.504-.34.984 2.254.395 3.785 1.27 3.867 1.317a.8.8 0 1 1-.805 1.382C17.176 7.68 14.93 6.398 12 6.398c-2.93 0-5.176 1.282-5.2 1.293a.8.8 0 0 1-.805-1.383c.083-.046 1.622-.925 3.88-1.32-.172-.484-.348-.972-.352-.98a.487.487 0 0 0-.484-.285c-.129.011-3.195.273-5.16 1.855C2.852 6.528.8 12.074.8 16.871c0 .082.02.164.062.238 1.418 2.489 5.282 3.141 6.16 3.168h.016c.156 0 .3-.074.395-.199l.949-1.289c-2.086-.504-3.192-1.293-3.258-1.344a.799.799 0 0 1-.168-1.117.794.794 0 0 1 1.113-.172c.032.016 2.067 1.446 5.93 1.446 3.879 0 5.91-1.434 5.93-1.45a.8.8 0 0 1 .945 1.293c-.066.047-1.164.836-3.246 1.34l.937 1.293c.094.125.239.2.395.2h.016c.882-.028 4.742-.68 6.16-3.169a.477.477 0 0 0 .062-.242c0-4.793-2.05-10.34-3.101-11.308zM8.8 15.199c-.887 0-1.602-.894-1.602-2 0-1.105.715-2 1.602-2 .883 0 1.597.895 1.597 2 0 1.106-.714 2-1.597 2zm6.398 0c-.883 0-1.597-.894-1.597-2 0-1.105.714-2 1.597-2 .887 0 1.602.895 1.602 2 0 1.106-.715 2-1.602 2zm0 0"/>
        </symbol>

        <symbol id="service_twitch" viewBox="0 0 24 24" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M4.8 3.2L3.2 6.397V19.2h4v2.403h3.198l2.403-2.403H16l4.8-4.8v-11.2zm14.4 10.402L16.8 16H12l-2.398 2.398V16H6.398V4.8H19.2zm0 0" /><path d="M15.2 12.8h-1.598V7.2h1.597zm-3.2 0h-1.602V7.2H12zm0 0" />
        </symbol>

        <symbol id="service_messenger" viewBox="0 0 24 24" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M5.602 22.398L10.398 20l-4.796-2.398zm0 0" /><path d="M12 2.398c-5.3 0-9.602 4.122-9.602 9.204C2.398 16.68 6.7 20.8 12 20.8c5.3 0 9.602-4.121 9.602-9.2 0-5.081-4.301-9.203-9.602-9.203zm.91 11.844l-2.305-2.48-4.328 2.422 4.813-5.098 2.36 2.363 4.218-2.363zm0 0" />
        </symbol>

        <symbol id="service_snapchat" viewBox="0 0 24 24" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M12.176 4c.715 0 3.136.191 4.277 2.668.383.828.285 2.273.211 3.437l-.004.051c-.008.164-.02.32-.027.469.015.02.164.156.492.168.25-.012.54-.086.855-.23a.784.784 0 0 1 .57.008h.005c.254.09.422.261.425.44.004.173-.128.43-.789.68a2.694 2.694 0 0 1-.25.082c-.375.118-.945.293-1.117.692-.097.215-.066.48.09.785 0 .004.004.008.004.012.047.105 1.187 2.62 3.73 3.027.094.016.16.094.153.188a.24.24 0 0 1-.024.101c-.105.238-.578.574-2.234.824-.133.02-.188.188-.266.547-.03.13-.058.258-.101.39-.035.118-.11.173-.235.173h-.02a2.34 2.34 0 0 1-.37-.043 4.986 4.986 0 0 0-.996-.102c-.23 0-.473.02-.715.059-.496.078-.918.367-1.363.672-.653.445-1.32.902-2.364.902-.047 0-.09 0-.136-.004-.028.004-.055.004-.086.004-1.043 0-1.711-.457-2.36-.902-.445-.305-.867-.594-1.363-.672a4.533 4.533 0 0 0-.719-.059c-.418 0-.75.063-.992.106a2.02 2.02 0 0 1-.371.054c-.102 0-.211-.023-.258-.18-.039-.136-.07-.269-.101-.394-.075-.328-.125-.531-.266-.55-1.656-.247-2.129-.587-2.234-.825-.012-.035-.024-.066-.024-.101a.182.182 0 0 1 .156-.188c2.54-.406 3.68-2.922 3.727-3.031.004 0 .004-.004.004-.008.156-.305.187-.57.094-.79-.176-.398-.747-.57-1.122-.687a3.147 3.147 0 0 1-.25-.082c-.75-.289-.812-.582-.785-.734.051-.254.407-.434.692-.434a.49.49 0 0 1 .207.04c.336.152.64.23.906.23.363 0 .52-.148.54-.168-.009-.164-.02-.34-.032-.52-.074-1.164-.168-2.609.21-3.433 1.138-2.477 3.555-2.668 4.27-2.668L12.133 4h.043m0-1.602h-.043l-.313.008v-.004c-.953 0-4.187.262-5.722 3.598-.387.844-.45 1.887-.422 2.922-.922.02-2 .625-2.215 1.726-.082.407-.184 1.786 1.781 2.54.012.003.02.007.031.011-.39.559-1.113 1.34-2.168 1.508-.902.14-1.55.941-1.5 1.86.016.226.067.44.153.64.41.938 1.406 1.363 2.543 1.613a1.83 1.83 0 0 0 1.785 1.305c.246 0 .465-.043.66-.078a3.44 3.44 0 0 1 .703-.082c.149 0 .305.012.465.039.14.023.457.238.711.41.73.5 1.727 1.184 3.266 1.184h.101c.04 0 .078.004.121.004 1.532 0 2.528-.68 3.258-1.176.281-.192.582-.399.723-.422.156-.024.312-.04.46-.04.259 0 .458.032.696.075.266.05.477.074.668.074.852 0 1.543-.508 1.785-1.293 1.137-.25 2.129-.672 2.535-1.593.094-.22.149-.43.16-.649a1.783 1.783 0 0 0-1.496-1.871c-1.054-.168-1.78-.95-2.172-1.508l.036-.011c1.601-.618 1.824-1.645 1.816-2.204-.02-.855-.594-1.601-1.477-1.918a2.37 2.37 0 0 0-.777-.156c.027-1.015-.039-2.078-.422-2.914-1.539-3.336-4.773-3.598-5.73-3.598zm0 0" />
        </symbol>

        <symbol id="service_twitter" viewBox="0 0 24 24" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M22.398 5.55a8.583 8.583 0 0 1-2.449.673 4.252 4.252 0 0 0 1.875-2.364 8.66 8.66 0 0 1-2.71 1.04A4.251 4.251 0 0 0 16 3.546a4.27 4.27 0 0 0-4.266 4.27c0 .335.036.66.11.972a12.126 12.126 0 0 1-8.797-4.46 4.259 4.259 0 0 0-.578 2.148c0 1.48.754 2.785 1.898 3.55a4.273 4.273 0 0 1-1.933-.535v.055a4.27 4.27 0 0 0 3.425 4.183c-.359.098-.734.149-1.125.149-.273 0-.543-.027-.804-.074a4.276 4.276 0 0 0 3.988 2.965 8.562 8.562 0 0 1-5.3 1.824 8.82 8.82 0 0 1-1.02-.059 12.088 12.088 0 0 0 6.543 1.918c7.851 0 12.14-6.504 12.14-12.144 0-.184-.004-.368-.011-.551a8.599 8.599 0 0 0 2.128-2.207zm0 0" />
        </symbol>

        <symbol id="service_instagram" viewBox="0 0 24 24" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M12 8.8A3.2 3.2 0 0 0 8.8 12a3.2 3.2 0 0 0 3.2 3.2 3.2 3.2 0 0 0 3.2-3.2A3.2 3.2 0 0 0 12 8.8zm0 0" /><path d="M16 2.398H8A5.609 5.609 0 0 0 2.398 8v8A5.609 5.609 0 0 0 8 21.602h8A5.609 5.609 0 0 0 21.602 16V8A5.609 5.609 0 0 0 16 2.398zm-4 14.403A4.805 4.805 0 0 1 7.2 12c0-2.648 2.152-4.8 4.8-4.8 2.648 0 4.8 2.152 4.8 4.8 0 2.648-2.152 4.8-4.8 4.8zm5.602-9.602a.799.799 0 1 1 0 0zm0 0" />
        </symbol>

        <symbol id="service_whatsapp" viewBox="0 0 24 24" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M3.836 16.668l-1.352 4.934 5.047-1.329zm0 0" /><path d="M12 2.398C6.7 2.398 2.398 6.7 2.398 12c0 5.3 4.301 9.602 9.602 9.602 5.3 0 9.602-4.301 9.602-9.602 0-5.3-4.301-9.602-9.602-9.602zm4.738 12.915c-.195.554-1.168 1.093-1.601 1.128-.442.043-.852.2-2.856-.59-2.418-.953-3.945-3.433-4.062-3.593-.121-.156-.969-1.285-.969-2.453 0-1.172.613-1.746.828-1.985a.875.875 0 0 1 .637-.297c.156 0 .316 0 .453.004.172.004.36.016.535.41.215.47.676 1.645.735 1.766.058.117.101.262.019.418-.078.156-.121.254-.234.399-.121.136-.25.308-.36.41-.117.12-.242.25-.101.488.136.238.613 1.016 1.32 1.645.906.812 1.672 1.062 1.91 1.18.238.12.38.1.516-.06.14-.156.594-.69.754-.93.156-.237.316-.198.531-.12.219.078 1.39.656 1.629.773.238.121.394.18.453.278.063.097.063.574-.137 1.129zm0 0" />
        </symbol>

        <symbol id="service_facebook" viewBox="0 0 27 27" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M12 0C5.371 0 0 5.371 0 12c0 6.016 4.434 10.984 10.207 11.852V15.18H7.238v-3.153h2.969V9.926c0-3.473 1.691-5 4.578-5 1.387 0 2.117.105 2.461.148v2.754h-1.969c-1.226 0-1.652 1.164-1.652 2.473v1.726h3.594l-.489 3.153h-3.105v8.699C19.48 23.082 24 18.074 24 12c0-6.629-5.371-12-12-12zm0 0" />
        </symbol>

        <symbol id="service_netflix" viewBox="0 0 450 600" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M83.5 72.814V512l17.432-2.865a955.35 955.35 0 0 1 88.604-10.312l13.965-.966V338.206L83.5 72.814z"/><path d="M308.5 0L308.5 172.328 428.5 438.914 428.5 0z"/><path d="M308.5 245.415l-10.87-24.149L198.03 0H83.501l168.12 371.813 57.024 126.112 8.852.566a955.65 955.65 0 0 1 93.572 10.644L428.5 512l-120-266.585z"/>
        </symbol>

        <symbol id="service_vk" viewBox="0 0 24 24" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M12 .96C5.914.96.96 5.915.96 12c0 6.086 4.954 11.04 11.04 11.04 6.086 0 11.04-4.954 11.04-11.04C23.04 5.914 18.085.96 12 .96zm4.785 13.216c1.074.953 1.3 1.293 1.336 1.351.445.707-.492.793-.492.793h-1.98s-.481.004-.891-.27c-.672-.437-1.375-1.288-1.867-1.14-.414.125-.41.684-.41 1.16 0 .172-.149.25-.481.25h-.617c-1.086 0-2.262-.363-3.434-1.59-1.656-1.734-3.113-5.222-3.113-5.222s-.086-.176.008-.281c.105-.122.394-.106.394-.106h1.918s.18.031.309.125c.11.074.168.219.168.219s.32 1.062.734 1.742c.801 1.32 1.172 1.355 1.445 1.215.399-.207.266-1.617.266-1.617s.02-.602-.187-.871c-.16-.211-.465-.32-.598-.336-.11-.016.07-.203.3-.313.31-.137.727-.172 1.446-.164.563.004.723.04.941.09.665.152.5.555.5 1.969 0 .453-.062 1.09.278 1.3.148.09.652.204 1.55-1.257.43-.692.77-1.84.77-1.84s.067-.125.176-.188c.113-.066.11-.062.262-.062.152 0 1.683-.012 2.02-.012.335 0 .651-.004.702.191.078.282-.246 1.25-1.07 2.305-1.355 1.723-1.504 1.563-.383 2.559zm0 0" />
        </symbol>

        <symbol id="service_ok" viewBox="0 0 96 96" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M50 28c-3.313 0-6 2.688-6 6 0 3.313 2.688 6 6 6 3.313 0 6-2.688 6-6 0-3.313-2.688-6-6-6zm0 0" /><path d="M50 4C24.637 4 4 24.637 4 50s20.637 46 46 46 46-20.637 46-46S75.363 4 50 4zm0 16c7.73 0 14 6.27 14 14s-6.27 14-14 14-14-6.27-14-14 6.27-14 14-14zm14.828 49.172A3.999 3.999 0 0 1 62 76a3.987 3.987 0 0 1-2.828-1.172L50 65.656l-9.172 9.172a3.999 3.999 0 0 1-5.656 0 3.999 3.999 0 0 1 0-5.656l6.43-6.43c-1.836-.539-3.618-1.207-5.29-2.066A4.302 4.302 0 0 1 34 56.859c0-2.98 3.172-4.761 5.809-3.375A21.767 21.767 0 0 0 50 56c3.684 0 7.148-.91 10.191-2.516C62.828 52.098 66 53.88 66 56.86c0 1.602-.89 3.078-2.313 3.813-1.671.863-3.453 1.531-5.289 2.07zm0 0" />
        </symbol>

        <symbol id="service_steam" viewBox="0 0 22 22" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M14.398 7.2a2.4 2.4 0 1 0 .003 4.799 2.4 2.4 0 0 0-.003-4.8zm0 0" fill="none" strokeWidth="1.6" stroke="currentColor" strokeMiterlimit="10"/><path d="M8 14c-.629 0-1.18.297-1.547.75l1.758.48c.426.114.68.555.562.98a.804.804 0 0 1-.984.563l-1.762-.48A1.998 1.998 0 0 0 10 16c0-1.105-.895-2-2-2zm0 0" /><path d="M19.2 3.2H4.8c-.886 0-1.6.714-1.6 1.6v9.063l2.027.551a3.213 3.213 0 0 1 2.289-1.566l2.136-2.567a4.799 4.799 0 1 1 4.066 4.066l-2.566 2.137A3.195 3.195 0 0 1 8 19.2 3.2 3.2 0 0 1 4.8 16c0-.016.005-.027.005-.043l-1.606-.437v3.68c0 .886.715 1.6 1.602 1.6h14.398c.887 0 1.602-.714 1.602-1.6V4.8c0-.886-.715-1.6-1.602-1.6zm0 0" />
        </symbol>

        <symbol id="service_epic_games" viewBox="0 0 50 50" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M 10 3 C 6.69 3 4 5.69 4 9 L 4 41.240234 L 25 47.539062 L 46 41.240234 L 46 9 C 46 5.69 43.31 3 40 3 L 10 3 z M 11 8 L 15 8 L 15 11 L 11 11 L 11 18 L 14 18 L 14 21 L 11 21 L 11 28 L 15 28 L 15 31 L 11 31 C 9.34 31 8 29.66 8 28 L 8 11 C 8 9.34 9.34 8 11 8 z M 17 8 L 23 8 C 24.66 8 26 9.34 26 11 L 26 18 C 26 19.66 24.66 21 23 21 L 20 21 L 20 31 L 17 31 L 17 8 z M 28 8 L 31 8 L 31 31 L 28 31 L 28 8 z M 36 8 L 39 8 C 40.66 8 42 9.34 42 11 L 42 15 L 39 15 L 39 11 L 36 11 L 36 28 L 39 28 L 39 24 L 42 24 L 42 28 C 42 29.66 40.66 31 39 31 L 36 31 C 34.34 31 33 29.66 33 28 L 33 11 C 33 9.34 34.34 8 36 8 z M 20 11 L 20 18 L 23 18 L 23 11 L 20 11 z M 9 34 L 13 34 C 13.55 34 14 34.45 14 35 L 14 36 L 13 36 L 13 35.25 C 13 35.11 12.89 35 12.75 35 L 9.25 35 C 9.11 35 9 35.11 9 35.25 L 9 38.75 C 9 38.89 9.11 39 9.25 39 L 12.75 39 C 12.89 39 13 38.89 13 38.75 L 13 38 L 12 38 L 12 37 L 14 37 L 14 39 C 14 39.55 13.55 40 13 40 L 9 40 C 8.45 40 8 39.55 8 39 L 8 35 C 8 34.45 8.45 34 9 34 z M 18 34 L 19 34 L 22 40 L 21 40 L 20.5 39 L 16.5 39 L 16 40 L 15 40 L 18 34 z M 23 34 L 24 34 L 26 38 L 28 34 L 29 34 L 29 40 L 28 40 L 28 36 L 26.5 39 L 25.5 39 L 24 36 L 24 40 L 23 40 L 23 34 z M 30 34 L 35 34 L 35 35 L 31 35 L 31 36.5 L 33 36.5 L 33 37.5 L 31 37.5 L 31 39 L 35 39 L 35 40 L 30 40 L 30 34 z M 37 34 L 41 34 C 41.55 34 42 34.45 42 35 L 42 35.5 L 41 35.5 L 41 35.25 C 41 35.11 40.89 35 40.75 35 L 37.25 35 C 37.11 35 37 35.11 37 35.25 L 37 36.25 C 37 36.39 37.11 36.5 37.25 36.5 L 41 36.5 C 41.55 36.5 42 36.95 42 37.5 L 42 39 C 42 39.55 41.55 40 41 40 L 37 40 C 36.45 40 36 39.55 36 39 L 36 38.5 L 37 38.5 L 37 38.75 C 37 38.89 37.11 39 37.25 39 L 40.75 39 C 40.89 39 41 38.89 41 38.75 L 41 37.75 C 41 37.61 40.89 37.5 40.75 37.5 L 37 37.5 C 36.45 37.5 36 37.05 36 36.5 L 36 35 C 36 34.45 36.45 34 37 34 z M 18.5 35 L 17 38 L 20 38 L 18.5 35 z"></path>
        </symbol>

        <symbol id="service_skype" viewBox="0 0 26 26" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M23.363 14.387c.153-.739.23-1.5.23-2.266C23.594 5.883 18.45.805 12.122.805c-.594 0-1.191.047-1.781.136A6.891 6.891 0 0 0 6.852 0C3.074 0 0 3.035 0 6.762c0 1.144.293 2.27.852 3.265-.133.688-.2 1.391-.2 2.094 0 6.238 5.149 11.316 11.47 11.316.648 0 1.3-.054 1.94-.164.95.477 2.012.727 3.086.727C20.926 24 24 20.969 24 17.238c0-1.004-.215-1.96-.637-2.851zM17.758 17.3c-.508.707-1.258 1.27-2.23 1.668-.966.394-2.122.593-3.434.593-1.578 0-2.903-.273-3.934-.812a5.074 5.074 0 0 1-1.808-1.582c-.47-.664-.707-1.324-.707-1.961 0-.395.156-.738.457-1.023.304-.278.687-.418 1.148-.418.379 0 .703.109.969.332.254.21.469.523.644.93.192.437.407.808.633 1.1.211.282.524.52.918.704.399.188.938.281 1.598.281.91 0 1.652-.191 2.215-.57.546-.367.812-.813.812-1.352 0-.43-.14-.765-.422-1.027-.3-.277-.699-.492-1.176-.637-.5-.152-1.18-.32-2.015-.496-1.14-.238-2.11-.523-2.88-.847-.788-.332-1.425-.79-1.89-1.364-.472-.582-.71-1.312-.71-2.172 0-.816.253-1.554.75-2.191.488-.633 1.206-1.125 2.132-1.46.91-.333 1.996-.5 3.223-.5.98 0 1.844.108 2.566.331.723.223 1.336.524 1.813.89.484.376.843.774 1.07 1.188.227.418.344.832.344 1.235 0 .386-.153.738-.453 1.046-.297.31-.68.465-1.125.465-.41 0-.727-.097-.95-.289-.207-.18-.418-.46-.656-.863-.273-.516-.605-.918-.984-1.203-.371-.277-.989-.418-1.836-.418-.79 0-1.43.156-1.902.465-.461.293-.684.633-.684 1.039 0 .246.07.449.219.629.156.187.379.351.656.488.289.145.586.258.883.34.308.082.82.207 1.523.367.887.191 1.707.398 2.43.625.73.234 1.363.516 1.879.848.527.34.941.773 1.238 1.293.297.52.445 1.16.445 1.91a4.07 4.07 0 0 1-.77 2.418zm0 0"/>
        </symbol>

        <symbol id="service_mail_ru" viewBox="0 0 512 512" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M256 141.176c-63.306 0-114.809 51.503-114.809 114.809S192.694 370.795 256 370.795s114.809-51.503 114.809-114.809S319.306 141.176 256 141.176zm0 188.254c-40.498 0-73.445-32.947-73.445-73.445 0-40.498 32.947-73.445 73.445-73.445 40.499 0 73.445 32.947 73.445 73.445 0 40.498-32.946 73.445-73.445 73.445z"/><path d="M437.008 74.97C388.656 26.623 324.375 0 256 0h-.017C187.603.004 123.318 26.637 74.97 74.992 26.62 123.347-.005 187.637 0 256.017c.004 68.379 26.637 132.666 74.992 181.014C123.344 485.377 187.625 512.001 256 512h.017c55.945-.004 111.216-18.738 155.631-52.752 9.07-6.945 10.792-19.927 3.846-28.995-6.945-9.069-19.926-10.794-28.995-3.846-37.24 28.518-83.58 44.224-130.486 44.228h-.014c-57.324 0-111.224-22.324-151.761-62.856-40.542-40.536-62.871-94.435-62.875-151.766-.006-118.35 96.273-214.641 214.623-214.649H256c118.34 0 214.628 96.279 214.636 214.622v23.532c0 27.523-22.39 49.913-49.913 49.913-27.523 0-49.913-22.391-49.913-49.913v-23.532c0-11.422-9.259-20.682-20.682-20.682s-20.682 9.26-20.682 20.682v23.532c0 50.33 40.947 91.278 91.278 91.278S512 329.848 512 279.518v-23.534c-.005-68.38-26.638-132.666-74.992-181.014z"/>
        </symbol>

        <symbol id="service_tiktok" viewBox="0 0 50 50" fill="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <path d="M41 4H9C6.243 4 4 6.243 4 9v32c0 2.757 2.243 5 5 5h32c2.757 0 5-2.243 5-5V9c0-2.757-2.243-5-5-5zm-3.994 18.323a7.482 7.482 0 0 1-.69.035 7.492 7.492 0 0 1-6.269-3.388v11.537a8.527 8.527 0 1 1-8.527-8.527c.178 0 .352.016.527.027v4.202c-.175-.021-.347-.053-.527-.053a4.351 4.351 0 1 0 0 8.704c2.404 0 4.527-1.894 4.527-4.298l.042-19.594h4.016a7.488 7.488 0 0 0 6.901 6.685v4.67z"/>
        </symbol>

        <symbol id="question" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <circle cx="12" cy="12" r="10" /><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" /><line x1="12" y1="17" x2="12" y2="17" />
        </symbol>

        <symbol id="question" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2">
            <circle cx="12" cy="12" r="10" /><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" /><line x1="12" y1="17" x2="12" y2="17" />
        </symbol>

        <symbol id="network" viewBox="0 0 50 50" fill="currentColor" strokeLinecap="round" strokeLinejoin="round">
            <path d="M 25 7 C 15.941406 7 7.339844 10.472656 0.78125 16.773438 L 0.0625 17.464844 L 5.59375 23.230469 L 6.320313 22.539063 C 11.378906 17.679688 18.015625 15 25 15 C 31.984375 15 38.621094 17.679688 43.683594 22.539063 L 44.40625 23.230469 L 49.941406 17.464844 L 49.21875 16.769531 C 42.660156 10.46875 34.058594 7 25 7 Z M 25 19 C 19.046875 19 13.394531 21.28125 9.085938 25.421875 L 8.363281 26.113281 L 13.921875 31.90625 L 14.644531 31.210938 C 17.464844 28.496094 21.144531 27 25 27 C 28.855469 27 32.535156 28.496094 35.355469 31.210938 L 36.078125 31.90625 L 41.636719 26.113281 L 40.917969 25.421875 C 36.605469 21.28125 30.953125 19 25 19 Z M 25 31 C 22.15625 31 19.453125 32.089844 17.390625 34.074219 L 16.671875 34.765625 L 25 43.441406 L 33.328125 34.765625 L 32.609375 34.074219 C 30.546875 32.089844 27.84375 31 25 31 Z"/>
        </symbol>

        <symbol id="location" viewBox="0 0 24 24" fill="currentColor" strokeLinecap="round" strokeLinejoin="round">
            <path d="M12,2C8.134,2,5,5.134,5,9c0,5,7,13,7,13s7-8,7-13C19,5.134,15.866,2,12,2z M12,11.5c-1.381,0-2.5-1.119-2.5-2.5 c0-1.381,1.119-2.5,2.5-2.5s2.5,1.119,2.5,2.5C14.5,10.381,13.381,11.5,12,11.5z"/>
        </symbol>
    </svg>
);

export default Icons;
