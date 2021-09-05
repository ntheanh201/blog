
call deno run --allow-read --allow-write .\process.js
@rem call npm i -g sirv-cli
start "" http://localhost:5000/
call sirv --dev --host
