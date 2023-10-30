var zip = require('bestzip');
var packageJson = require('./package.json');

zip({
  source: packageJson.name + '/*',
  destination: './' + packageJson.name + '-' + packageJson.version + '.zip'
}).then(function() {
  console.log('Zip complete.');
}).catch(function(err) {
  console.error(err.stack);
  process.exit(1);
});

zip({
  source: packageJson.name + '-full' + '/*',
  destination: './' + packageJson.name + '-full' + '-' + packageJson.version + '.zip'
}).then(function() {
  console.log('Zip complete.');
}).catch(function(err) {
  console.error(err.stack);
  process.exit(1);
});