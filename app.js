const http = require('http'),
	crypto = require('crypto'),
	express = require('express'),
	compression = require('compression'),
	bodyParser = require('body-parser'),
	rateLimiter = require('express-rate-limit'),
	mongo = require('mongodb').MongoClient;

const police = require('./police/index'),
	db = require('./db/db');

const app = express();

const MONGODB_URI = process.env.MONGODB_URI || 'mongodb://localhost:27017/envelope',
	PORT = process.env.PORT || 5000;

mongo.connect(MONGODB_URI, (err, db) => {
	if (err) {
		console.error(`Error in connecting to DB: ${err}`);
		process.exit(1);
	}

	app.locals.db = db;

	const posts = db.collection('posts');

	posts.createIndex({
		share_hash: 1
	}, {
		unique: true
	});


	http.createServer(app).listen(PORT, err => {
		if (err) {
			console.error(`Error in starting server: ${err.message}`);
			process.exit(1);
		}

		console.log(`Server started on http://localhost:${PORT}`);
	});
});

const rateLimit = new rateLimiter({
	windowMs: 60 * 60 * 1000,
	max: 15,
	delayMs: 0
});

app.enable('trust proxy');

// Enable compression
app.use(compression());

// Enable body parser
app.use(bodyParser.urlencoded({
	extended: true
}));

// Rate limit the following endpoints

app.use('/submit-photo', rateLimit);

app.post('/submit-post', (req, res, next) => {

	if (req.body && req.body.text) {
		next();
	} else {
		sendErrorResponse(req, res, {
			message: 'Bad Request',
			code: 400
		}, 400);
	}

}, rateLimit, (req, res) => {
	const mongoDB = req.app.locals.db;
	const posts = mongoDB.collection('posts');

	// Share hash will be used in the link that is generated when user shares a post 
	// The link will have share_hash in it and when someone visits that page they'll 
	// see the shared post
	let shareHash = createShareHash();
	// Edit hash is generated and given to the original poster
	// The only possible way to edit the post will be to have the edit link(which contains edit_hash) with you
	// if you lose the edit link then no one can edit the post
	let editHash = createEditHash();
	// Time in unix format
	const time = Math.floor(new Date() / 1000);

	const md5Hash = createMD5Hash(req.body.text);

	let postdata = {
		share_hash: shareHash,
		text: req.body.text,
		time: time,
		edit_hash: editHash,
		md5: md5Hash
	};

	db.save(postdata, posts).then(result => {
		console.log(`Saved ${result.result.ok} post`);
		res.send(JSON.stringify({
			share_hash: postdata.share_hash,
			edit_hash: postdata.edit_hash,
			time: postdata.time,
		}));

	}, err => {
		if (err.code == '11000') {
			console.error(`duplicate post. md5: ${postdata.md5}, share_hash: ${postdata.share_hash}`);

			sendErrorResponse(req, res, {
				message: err.message,
				code: err.code
			}, 400);

		} else {
			console.error(`Error occurred in storing post: ${err}`);
			sendErrorResponse(req, res, {
				message: err.message,
				code: err.code
			}, 500);
		}
	});
});

function createShareHash() {
	return crypto.randomBytes(8).toString('hex');
}

function createEditHash() {
	return crypto.randomBytes(32).toString('hex');
}

function createMD5Hash(text) {
	return crypto.createHash('md5').update(text).digest('hex');
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