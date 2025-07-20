const MsgTypeUnknown = 0;
const MsgTypeError = 1;
const MsgTypeLogLine = 2;
const MsgTypeActionStart = 3;
const MsgTypeActionComplete = 4;
const MsgTypeFileFetch = 5;

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
            }).showToast();
            break;
        case MsgTypeLogLine:
            console.log(msg.Message);
            break;
        case MsgTypeActionStart:
            console.log(msg.Message);
            Toastify({
                text: "Started Action: " + msg.Action + " (" + msg.Message + ")",
                duration: 3000
            }).showToast();
            break;
        case MsgTypeActionComplete:
            Toastify({
                text: "Completed Action: " + msg.Action,
                duration: 3000
            }).showToast();
            break;
        case MsgTypeFileFetch:
            console.log("fetched file:", msg.Filename);
            Toastify({
                text: "Fetched file: " + msg.Filename,
                duration: 3000
            }).showToast();
            break;
        }

    } catch (error) {
        console.error("Error parsing JSON:", error);
        console.log("Received data (unparsed):", event.data);
    }
});
