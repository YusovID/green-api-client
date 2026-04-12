"use strict";

const $ = (sel) => document.getElementById(sel);

const responseBox = $("response");

function creds() {
  return {
    idInstance: $("idInstance").value.trim(),
    apiTokenInstance: $("apiTokenInstance").value.trim(),
  };
}

function phoneToChatId(phone) {
  return phone.replace(/\D/g, "") + "@c.us";
}

async function callAPI(endpoint, body) {
  responseBox.value = "Загрузка...";

  try {
    const res = await fetch("/api/" + endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });

    const data = await res.json();
    responseBox.value = JSON.stringify(data, null, 2);
  } catch (err) {
    responseBox.value = "Ошибка: " + err.message;
  }
}

const actions = {
  getSettings() {
    callAPI("getSettings", creds());
  },

  getStateInstance() {
    callAPI("getStateInstance", creds());
  },

  sendMessage() {
    callAPI("sendMessage", {
      ...creds(),
      chatId: phoneToChatId($("msgPhone").value),
      message: $("msgText").value,
    });
  },

  sendFileByUrl() {
    callAPI("sendFileByUrl", {
      ...creds(),
      chatId: phoneToChatId($("filePhone").value),
      urlFile: $("fileUrl").value.trim(),
    });
  },
};

document.addEventListener("click", (e) => {
  const action = e.target.dataset?.action;
  if (action && actions[action]) {
    actions[action]();
  }
});
