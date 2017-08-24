function save(data, coll) {

	return new Promise((onResolve, onReject) => {
		coll.insertOne(data, (err, result) => {
			if (err) {
				onReject(err);
			}

			if (result) {
				onResolve(result);
			}
		});
	});
}




module.exports = {
	save: save
};