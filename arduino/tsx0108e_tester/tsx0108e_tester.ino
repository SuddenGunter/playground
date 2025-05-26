const int pinB1 = 2;   // drives B1 side
const int pinA1 = 3;   // reads A1 side

void setup() {
  pinMode(pinB1, OUTPUT);
  pinMode(pinA1, INPUT);
  Serial.begin(115200);
}

void loop() {
  digitalWrite(pinB1, HIGH);
  delay(1000);
  Serial.print("A1 reads: ");
  Serial.println(digitalRead(pinA1));   // expect HIGH (3.3 V)
  delay(1000);

  digitalWrite(pinB1, LOW);
  delay(1000);
  Serial.print("A1 reads: ");
  Serial.println(digitalRead(pinA1));   // expect LOW (0 V)
  delay(1000);
}