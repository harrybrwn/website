const tooltip = document.getElementById("cp-id-tooltip");
const btn = tooltip.children[0]
const defaultTooltipMsg = "Copy ID";

function copyTooltip(text, tooltip, msgTxt) {
  const button = tooltip.children[0];
  tooltip.setAttribute("data-text", msgTxt ? msgTxt : "Copy");
  button.addEventListener("click", function() {
    // navigator.clipboard.writeText(text);
    navigator.clipboard.writeText(button.innerText);
    tooltip.setAttribute("data-text", "Copied!");
  });
  button.addEventListener("mouseout", function() {
    tooltip.setAttribute("data-text", "Copy");
  });
}

copyTooltip("{{ id }}", document.getElementById("cp-id"), "Copy ID");
copyTooltip("{{ server }}/{{ id }}", document.getElementById("cp-link"), "Copy Link");
copyTooltip("{{ id }}", document.getElementById("cp-id-tooltip"));

const delBtn = document.getElementById("del-btn");
delBtn.addEventListener("click", function() {
  fetch("/{{ id }}", {
    method: "DELETE",
  });
});
