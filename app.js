const http = require('http'),
	crypto = require('crypto'),
	express = require('express'),
	compression = require('compression'),
	bodyParser = require('body-parser'),
	mongo = require('mongodb').MongoClient;

const police = require('./police/index'),
	router = require('./routes/routes'),
	submitRouter = require('./routes/submit-router'),
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

app.enable('trust proxy');

// Enable compression
app.use(compression());

// Enable body parser
app.use(bodyParser.urlencoded({
	extended: true
}));

app.use('/',router);

app.use('/submit', submitRouter);

// Rate limit the following endpoints

