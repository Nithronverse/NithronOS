import '@testing-library/jest-dom'

// Minimal DOM shims for tests
// @ts-ignore
window.alert = window.alert || (()=>{})
// @ts-ignore
window.confirm = window.confirm || (()=>true)

