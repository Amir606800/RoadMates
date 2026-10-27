export function buildDriverApplication(values) {
  return {
    contact_phone: values.phone.trim(), first_name: values.name.trim(), last_name: values.surname.trim(),
    pin_code: values.pin.trim().toUpperCase(), car_make: values.brand.trim(), car_model: values.model.trim(),
    car_year: Number(values.year), car_plate: values.plate.trim().toUpperCase(), car_color: values.color.trim(), seats: Number(values.seats),
  }
}
