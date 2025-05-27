#include <Wire.h>
#include <Adafruit_AHTX0.h>
#include <Adafruit_BMP280.h>

Adafruit_AHTX0 aht;
Adafruit_BMP280 bmp;

void setup() {
  Serial.begin(115200);
  delay(500);

  aht.begin();
  bmp.begin();
}

void loop() {
  sensors_event_t humidity, temperature;
  aht.getEvent(&humidity, &temperature);

  float bmpTemp = bmp.readTemperature();
  float pressure = bmp.readPressure() / 100.0;

  Serial.print("AHT20 - Temp:");
  Serial.println(temperature.temperature);
  Serial.print("AHT20 -  Humidity:");
  Serial.println(humidity.relative_humidity);
  Serial.print("BMP - Temp:");
  Serial.println(bmpTemp);
  Serial.print("BMP -  Pressure:");
  Serial.println(pressure);
  Serial.println("__");
  delay(3000);
}