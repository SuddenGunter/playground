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

    AHT20/BMP280 board:
    SDA/SLC to Arduino SDA/SLC.
    VCC pin is connected to Arduino 5V pin,
    GND pin is connected to Arduino GND pin,
 ****************************************************/


#include <Adafruit_GFX.h>
#include <Adafruit_ILI9341.h>
#include <avr/sleep.h>
#include <Wire.h>
#include <Adafruit_AHTX0.h>
#include <Adafruit_BMP280.h>

#define TFT_CS 8
#define TFT_RST 9
#define TFT_DC 10

Adafruit_ILI9341 tft = Adafruit_ILI9341(TFT_CS, TFT_DC, TFT_RST);
Adafruit_AHTX0 aht;
Adafruit_BMP280 bmp;

struct Measurement {
  float temp;
  float humidity;
  float pressure;
};

Measurement measure();
void display(const Measurement &m);
void haltForever();

void setup() {
  Serial.begin(115200);

  // Initialize sensors
  if (!aht.begin()) {
    Serial.println("Failed to initialize AHT20 sensor!");
    haltForever();
  }

  if (!bmp.begin()) {
    Serial.println("Failed to initialize BMP280 sensor!");
    haltForever();
  }

  bmp.setSampling(
    Adafruit_BMP280::MODE_NORMAL,
    Adafruit_BMP280::SAMPLING_X2,
    Adafruit_BMP280::SAMPLING_X4,
    Adafruit_BMP280::FILTER_X8,
    Adafruit_BMP280::STANDBY_MS_4000);

  tft.begin();
  tft.setRotation(0);
  tft.fillScreen(ILI9341_BLACK);
}


void loop(void) {
  Measurement m = measure();
  display(m);
  delay(3000);
}

Measurement measure() {
  sensors_event_t humidity, temperature;
  aht.getEvent(&humidity, &temperature);

  float pressure = bmp.readPressure() / 100.0;

  return Measurement{ temperature.temperature, humidity.relative_humidity, pressure };
}

void display(const Measurement &m) {
  unsigned long start = micros();
  tft.setCursor(0, 0);
  tft.setTextColor(ILI9341_WHITE, ILI9341_BLACK);
  tft.setTextSize(3);

  tft.print("T: ");
  tft.println(m.temp);
  tft.print("H: ");
  tft.println(m.humidity);
  tft.print("P: ");
  tft.println(m.pressure);
}


void haltForever() {
  set_sleep_mode(SLEEP_MODE_PWR_DOWN);  // Deep sleep
  sleep_enable();
  sleep_cpu();  // MCU sleeps here forever (unless reset or interrupt)
}
