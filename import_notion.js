const { Client } = require("@notionhq/client");

const notion = new Client({
  auth: "secret_Onq4b0gIoA7WldxN9tGdxl0zYyxA3y2CEhVDgiVb7yv",
});

const pageId = "568ac4c064c34ef6a6ad0b8d77230681";

async function doit() {
  let response = await notion.pages.retrieve({ page_id: pageId });
  // console.log(response);
  let blockId = response.id;
  response = await notion.blocks.children.list({
    block_id: blockId,
    page_size: 50,
  });
  console.log(response);
}

doit();
