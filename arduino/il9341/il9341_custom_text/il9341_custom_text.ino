/***************************************************
  Based on https://simple-circuit.com/interfacing-arduino-ili9341-tft-display/ and adafruit example.
  
  Arduino pins are 5v, while IL9341 expects 3.3v. To fix this I used txs0108e in between them

  Wiring :
    CS pin is connected to Arduino digital pin 8,
    RST pin is connected to Arduino digital pin 9,
    D/C pin is connected to Arduino digital pin 10,
    MOSI pin is connected to Arduino digital pin 11,
    SCK pin is connected to Arduino digital pin 13.

    Other pins are connected as follows:
    VCC pin is connected to Arduino 5V pin,
    GND pin is connected to Arduino GND pin,
    BL (LED) pin is connected to Arduino 5V pin,
    MISO pin is not connected.
 ****************************************************/


#include <Adafruit_GFX.h>       // include Adafruit graphics library
#include <Adafruit_ILI9341.h>   // include Adafruit ILI9341 TFT library
#include <avr/sleep.h>

#define TFT_CS    8      // TFT CS  pin is connected to arduino pin 8
#define TFT_RST   9      // TFT RST pin is connected to arduino pin 9
#define TFT_DC    10     // TFT DC  pin is connected to arduino pin 10
// initialize ILI9341 TFT library
Adafruit_ILI9341 tft = Adafruit_ILI9341(TFT_CS, TFT_DC, TFT_RST);

void setup() {
  Serial.begin(9600);
  Serial.println("ILI9341 Test!"); 
 
  tft.begin();
  tft.setRotation(0);
  testText();
}


void loop(void) {
    haltForever();
}

void haltForever() {
  set_sleep_mode(SLEEP_MODE_PWR_DOWN);  // Deep sleep
  sleep_enable();
  sleep_cpu();  // MCU sleeps here forever (unless reset or interrupt)
}

unsigned long testText() {
  tft.fillScreen(ILI9341_BLACK);
  unsigned long start = micros();
  tft.setCursor(0, 0);
  tft.setTextColor(ILI9341_WHITE);  tft.setTextSize(3);
  tft.println("Hello World!");
}
