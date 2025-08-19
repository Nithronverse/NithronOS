/* eslint-env node */
module.exports = {
	root: true,
	extends: [
		'eslint:recommended',
		'plugin:react-hooks/recommended',
		'plugin:react-refresh/recommended',
		'prettier',
	],
	parserOptions: {
		ecmaVersion: 'latest',
		sourceType: 'module',
	},
	env: {
		browser: true,
		es2020: true,
		node: true,
	},
	ignorePatterns: ['dist', 'node_modules'],
}


