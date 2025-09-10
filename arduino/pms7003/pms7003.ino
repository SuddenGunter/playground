#include <Plantower_PMS7003.h> // must use fork https://github.com/SuddenGunter/Plantower_PMS7003/tree/master otherwise will not compile
#include <SoftwareSerial.h>

char output[256];
Plantower_PMS7003 pms7003 = Plantower_PMS7003();
SoftwareSerial serial1(2, 3); // RX, TX

void setup()
{
  Serial.begin(9600);
  serial1.begin(9600);
  pms7003.init(&serial1);
  //pms7003.debug = true;

}
 
void loop()
{
  pms7003.updateFrame();

  if (pms7003.hasNewData()) {

    sprintf(output, "\nSensor Version: %d    Error Code: %d\n",
                  pms7003.getHWVersion(),
                  pms7003.getErrorCode());
    Serial.print(output);

    sprintf(output, "    PM1.0 (ug/m3): %2d     [atmos: %d]\n",
                  pms7003.getPM_1_0(),
                  pms7003.getPM_1_0_atmos());              
    Serial.print(output);
    sprintf(output, "    PM2.5 (ug/m3): %2d     [atmos: %d]\n",
                  pms7003.getPM_2_5(),
                  pms7003.getPM_2_5_atmos());
    Serial.print(output);
    sprintf(output, "    PM10  (ug/m3): %2d     [atmos: %d]\n",
                  pms7003.getPM_10_0(),
                  pms7003.getPM_10_0_atmos());              
    Serial.print(output);

    sprintf(output, "\n    RAW: %2d[>0.3] %2d[>0.5] %2d[>1.0] %2d[>2.5] %2d[>5.0] %2d[>10]\n",
                  pms7003.getRawGreaterThan_0_3(),
                  pms7003.getRawGreaterThan_0_5(),
                  pms7003.getRawGreaterThan_1_0(),
                  pms7003.getRawGreaterThan_2_5(),
                  pms7003.getRawGreaterThan_5_0(),
                  pms7003.getRawGreaterThan_10_0());
    Serial.print(output);


  }


}