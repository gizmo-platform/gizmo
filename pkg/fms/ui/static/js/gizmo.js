const MsgTypeUnknown = 0;
const MsgTypeError = 1;
const MsgTypeLogLine = 2;
const MsgTypeFileFetch = 3;

var ws = new ReconnectingWebSocket('ws://' + document.location.host + '/api/eventstream');

ws.addEventListener("message", (event) => {
    try {
        const msg = JSON.parse(event.data);

        switch (msg.Type) {
        case MsgTypeUnknown:
            console.error("Message type unknown!", msg);
            break;
        case MsgTypeError:
            console.error("Error from remote:", msg.Error);
            Toastify({
                text: "Error: " + msg.Error,
                duration: 8000,
                className: "status-error",
            }).showToast();
            break;
        case MsgTypeLogLine:
            console.log(msg.Message);
            break;
        case MsgTypeFileFetch:
            console.log("fetched file:", msg.Filename);
            Toastify({
                text: "Fetched file: " + msg.Filename,
                duration: 3000
            }).showToast();
        }

    } catch (error) {
        console.error("Error parsing JSON:", error);
        console.log("Received data (unparsed):", event.data);
    }
});
