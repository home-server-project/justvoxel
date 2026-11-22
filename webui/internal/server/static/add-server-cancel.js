(() => {
  let pending = false;
  const closeTab = () => {
    try {
      if (window.opener && !window.opener.closed &&
          window.opener.location.origin === window.location.origin) window.opener.focus();
    } catch {
      // An unavailable opener must not prevent closing.
    }
    try { window.close(); } catch {
      // Keep the cancelled state available when browser policy denies close.
    }
  };
  const showCancelled = () => {
    const main = document.createElement("main");
    for (const [tag, text] of [
      ["h1", jvT("ui.68817f9bc395")],
      ["p", jvT("ui.1f24ad373052")],
      ["p", jvT("ui.b3a62533c186")],
    ]) {
      const node = document.createElement(tag);
      node.textContent = text;
      main.appendChild(node);
    }
    const retry = document.createElement("button");
    retry.type = "button";
    retry.textContent = jvT("ui.50a3f9523122");
    retry.addEventListener("click", closeTab);
    main.appendChild(retry);
    document.body.className = "add-server-cancelled";
    document.body.replaceChildren(main);
    document.title = jvT("ui.771d7013ba7f");
    closeTab();
  };
  document.addEventListener("submit", async (event) => {
    const form = event.target;
    if (!(form instanceof HTMLFormElement)) return;
    const action = new URL(event.submitter?.getAttribute("formaction") || form.action, window.location.href);
    if (action.origin !== window.location.origin ||
        !["/minecraft/servers/add/cancel", "/minecraft/servers/add/review/cancel"].includes(action.pathname) ||
        form.method.toLowerCase() !== "post") return;
    event.preventDefault();
    if (pending) return;
    pending = true;
    const button = event.submitter;
    if (button) button.disabled = true;
    let error = form.querySelector("[data-add-server-cancel-error]");
    if (error) error.remove();
    try {
      const response = await fetch(action.href, {
        method: "POST", body: new FormData(form), credentials: "same-origin",
        headers: { Accept: "application/json" }, redirect: "error",
      });
      if (response.status !== 204) throw new Error(jvT("ui.59611746b055"));
      showCancelled();
    } catch {
      error = document.createElement("p");
      error.className = "notice error";
      error.dataset.addServerCancelError = "";
      error.setAttribute("role", "alert");
      error.textContent = jvT("ui.9ed7bd151e96");
      form.appendChild(error);
      pending = false;
      if (button) button.disabled = false;
    }
  });
})();
