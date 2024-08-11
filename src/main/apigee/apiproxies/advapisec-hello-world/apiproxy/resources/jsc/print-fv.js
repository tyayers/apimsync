 function delay(time) {
    var currentTime = new Date();
    var futureTime = new Date();
    while(futureTime.valueOf() < currentTime.valueOf() + time) {
        futureTime = new Date();
    }
}

delay(19 * 10);