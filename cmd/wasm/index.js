if (!WebAssembly.instantiateStreaming) {
  // polyfill
  WebAssembly.instantiateStreaming = async (resp, importObject) => {
    const source = await (await resp).arrayBuffer();
    return await WebAssembly.instantiate(source, importObject);
  };
}

const go = new Go();

let mod, inst;

WebAssembly.instantiateStreaming(fetch("lib.wasm"), go.importObject).then(
  result => {
    mod = result.module;
    inst = result.instance;
    document.getElementById("connectBtn").disabled = false;
  }
)
  .then(run);

async function run() {
  await go.run(inst);
  await reset();
}


async function connect() {
  var url = document.getElementById("bootstrap").value
  console.log("connect to ", url)
  StartQuorum(url).then(res => {
    console.log(res)
    document.getElementById("joinBtn").disabled = false;
  }).catch(err => console.error(err))
}

async function join() {
  var seed = document.getElementById("seed").value
  console.log("join to group", seed)
  JoinGroup(seed).then(res => console.log(res))
}

async function reset() {
  inst = await WebAssembly.instantiate(mod, go.importObject); // reset instance
}

