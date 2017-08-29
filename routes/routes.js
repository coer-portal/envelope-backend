const express = require('express'),
    router = express.Router();


router.post('/fetch', (req, res, next) => {
    if (req.body && (req.body.type == "old" || req.body.type == "new") && req.body.time && req.body.deviceid) {
        next();
    } else {
        sendErrorResponse(req, res, {
            message: 'Bad Request',
            code: 400
        }, 400);
    }
}, sendPosts);


// function for next middleware for sending posts
function sendPosts(req, res, next) {
    let fetch = {
        type: req.body.type,
        time: Number(req.body.time),
        count: req.body.count || 10,
        devId: req.body.deviceid
    }

    if (fetch.type == "new") {
        req.app.locals.db.collection("posts").find({ time: { $gte: fetch.time } }).limit(fetch.count).toArray(function (err, result) {
            if (err) {
                res.status(500).end();
                return console.error(`error in fetching posts. ${err}`);
            }

            res.send(JSON.stringify(result));
        });
    }
    if (fetch.type == "old") {
        req.app.locals.db.collection("posts").find({ time: { $lte: fetch.time } }).limit(fetch.count).toArray(function (err, result) {
            if (err) {
                res.status(500).end();
                return console.error(`error in fetching posts. ${err}`);
            }

            res.send(JSON.stringify(result));

        });
    }
}

function sendErrorResponse(req, res, err, code) {
    let accepts = req.headers['accept'];

    res.status(code);

    if (accepts.indexOf('*/*') != -1 || accepts.indexOf('application/json') != -1) {
        res.write(JSON.stringify({
            error: err.message,
            code: code
        }));
    } else {
        res.write(`<html>
        <head>
            <title>Error Occurred</title>
        </head>
        <body>
            <h1>Error Occurred: ${code}</h1>
            <p>${err.message}<p>
        </body>
    </html>`);
    }
    res.end();
}

module.exports = router;