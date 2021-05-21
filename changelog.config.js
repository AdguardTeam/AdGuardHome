module.exports = {
    "disableEmoji": true,
    "list": [
        "+ ",
        "* ",
        "- ",
    ],
    "maxMessageLength": 64,
    "minMessageLength": 3,
    "questions": [
        "type",
        "scope",
        "subject",
        "body",
        "issues",
    ],
    "scopes": [
        "",
        "ui",
        "global",
        "filtering",
        "home",
        "dnsforward",
        "dhcpd",
        "querylog",
        "documentation",
    ],
    "types": {
        "+ ": {
            "description": "A new feature",
            "emoji": "",
            "value": "+ "
        },
        "* ": {
            "description": "A code change that neither fixes a bug or adds a feature",
            "emoji": "",
            "value": "* "
        },
        "- ": {
            "description": "A bug fix",
            "emoji": "",
            "value": "- "
        }
    }
};
