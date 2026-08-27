#include <Arduino.h>
#include <SensirionI2CSgp41.h>
#include <NOxGasIndexAlgorithm.h>
#include <VOCGasIndexAlgorithm.h>

SensirionI2CSgp41 sgp41;
VOCGasIndexAlgorithm voc_algorithm;
NOxGasIndexAlgorithm nox_algorithm;

// Default ticks for 50% RH and 25°C
uint16_t defaultCompenRh = 0x8000;
uint16_t defaultCompenT  = 0x6666;

void setup() {
  Serial.begin(115200);
  while (!Serial);

  Wire.begin();
  sgp41.begin(Wire);

  // Note: Both algorithm instances default to 1-second sampling rate automatically
  Serial.println("SGP41 Initialized. Starting measurements...");
}

void loop() {
  uint16_t error;
  uint16_t srawVoc = 0;
  uint16_t srawNox = 0;

  error = sgp41.measureRawSignals(defaultCompenRh, defaultCompenT, srawVoc, srawNox);

  if (error == 0) {
    int32_t voc_index = voc_algorithm.process(srawVoc);
    int32_t nox_index = nox_algorithm.process(srawNox);

    Serial.print("Raw VOC: ");
    Serial.print(srawVoc);
    Serial.print(" | VOC Index: ");
    Serial.print(voc_index);
    
    Serial.print("  ===  Raw NOx: ");
    Serial.print(srawNox);
    Serial.print(" | NOx Index: ");
    Serial.println(nox_index);
  } else {
    Serial.println("Error reading SGP41 sensor data.");
  }

  delay(1000); 
}