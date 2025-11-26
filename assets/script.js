
let isProcessing = false;

// Хранилище для pending callbacks
window._pendingCallbacks = {};
window._callbackCounter = 0;

// Функция, которую вызывает Go для возврата результата
window._resolveCallback = function (callbackId, result) {
    if (window._pendingCallbacks[callbackId]) {
        window._pendingCallbacks[callbackId](result);
        delete window._pendingCallbacks[callbackId];
    }
};

// Обёртка для асинхронного вызова Go функции
function solveProblem(text, showLog) {
    return new Promise((resolve) => {
        const callbackId = 'cb_' + (++window._callbackCounter);
        window._pendingCallbacks[callbackId] = resolve;
        solveProblemAsync(text, showLog, callbackId);
    });
}

function typeWriter(text, elementId) {
    return new Promise((resolve) => {
        const element = document.getElementById(elementId);

        // Настройка скорости:
        // Чем длиннее текст, тем больше символов выводим за 1 раз.
        // Для коротких ответов - по 2 символа, для длинных - по 20+.
        let batchSize = 1;
        if (text.length > 50) batchSize = 3;
        if (text.length > 200) batchSize = 8;
        if (text.length > 500) batchSize = 25;
        if (text.length > 1000) batchSize = 50;

        let i = 0;

        function type() {
            if (i < text.length) {
                const chunk = text.slice(i, i + batchSize);

                element.textContent += chunk;

                i += batchSize;

                // Прокрутка вниз
                element.scrollTop = element.scrollHeight;

                // Рекурсия с минимальной задержкой
                setTimeout(type, 10);
            } else {
                resolve();
            }
        }
        type();
    });
}

async function processRequest() {
    const inputField = document.getElementById('input');
    const outputField = document.getElementById('output');
    const btn = document.getElementById('solveBtn');
    const inputText = inputField.value;

    if (!inputText) return;

    btn.classList.add('processing');
    btn.innerText = "PROCESSING...";
    outputField.textContent = ""; // Изменили текст
    inputField.disabled = true;
    btn.disabled = true;

    try {
        // Получаем состояние чекбокса
        const showLog = document.getElementById('showLog').checked;
        
        const response = await solveProblem(inputText, showLog);

        await typeWriter(response, 'output', 20);

    } catch (error) {
        outputField.textContent = "Error: " + error;
    } finally {
        btn.classList.remove('processing');
        btn.innerText = "SOLVE";
        inputField.disabled = false;
        btn.disabled = false;
        inputField.focus();
    }
}